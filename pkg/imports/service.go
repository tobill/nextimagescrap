package imports

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"io/fs"
	"log"
	"os"
	"regexp"
	"time"
)

type SourceFileRepository interface {
	GetSourceFiles(func(path string, info fs.DirEntry, err error) error) error
	GetSourceFile(fpath string) (*os.File, error)
}

type DestinationFileRepository interface {
	ExportToDirectory(media *SourceMedia, i int, ext string) error
}

type Service interface {
	ScanSourceDirectory() error
	DetectMimetype(force bool) error
	ComputeChecksums(force bool) error
	ExtractCreationDate(force bool) error
	OrganizeToFolder(force bool) error
}

type SourceDbRepository interface {
	AddFile(filename string) (string, error)
	HasFile(fileName string) (bool, error)
	GetFilesByMimetypeFilter(filter []string) ([]*SourceMedia, error)
	GetAllFiles() ([]*SourceMedia, error)
	SaveMedia(media *SourceMedia) (string, error)
	AddChecksum(media *SourceMedia) error
	GetAllCheckSum() (error, []*SourceChecksum)
	GetFileByKey(path string) (*SourceMedia, error)
}

type service struct {
	sfr SourceFileRepository
	sdr SourceDbRepository
	drf DestinationFileRepository
}

func NewService(sfr SourceFileRepository, sdr SourceDbRepository) Service {
	return &service{
		sfr: sfr,
		sdr: sdr,
	}
}

func NewOrganizeService(sfr SourceFileRepository, sdr SourceDbRepository, dfr DestinationFileRepository) Service {
	return &service{
		sfr: sfr,
		sdr: sdr,
		drf: dfr,
	}
}

func (s service) ScanSourceDirectory() error {
	err := s.sfr.GetSourceFiles(func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		hf, err := s.sdr.HasFile(path)
		if err != nil {
			return err
		}
		if hf == false {
			key, err := s.sdr.AddFile(path)
			if err != nil {
				return err
			}
			log.Printf("Added key %s", key)
		}
		return nil
	})
	return err
}

func (s service) ComputeChecksums(force bool) error {
	importFiles, err := s.sdr.GetAllFiles()

	if err != nil {
		return err
	}
	for index, entry := range importFiles {
		log.Printf("%v %v", index, entry.Checksum)
		if entry.Checksum != "" && force == false {
			log.Printf("checksum da")
			continue
		}
		fob, err := s.sfr.GetSourceFile(entry.Path)
		defer func(fob *os.File) {
			err = fob.Close()
		}(fob)
		if err != nil {
			return err
		}
		h := sha1.New()
		_, err = io.Copy(h, fob)
		if err != nil {
			return err
		}
		entry.Checksum = hex.EncodeToString(h.Sum(nil))
		//log.Printf("%s %s", entry.Path, entry.Checksum)
		_, err = s.sdr.SaveMedia(entry)
		err = s.sdr.AddChecksum(entry)
		if err != nil {
			return err
		}

	}
	return nil
}

func (s service) DetectMimetype(force bool) error {
	importFiles, err := s.sdr.GetAllFiles()

	if err != nil {
		return err
	}
	for index, entry := range importFiles {
		log.Printf("%v %v", index, entry)
		if entry.Mimetype != "" && force == false {
			continue
		}
		fob, err := os.Open(entry.Path)
		if err != nil {
			return err
		}
		b := make([]byte, 512)
		_, err = fob.Read(b)
		if err != nil {
			return err
		}
		mtype, err := mimetype.DetectReader(bytes.NewReader(b))
		log.Printf("%s %s", entry.Path, mtype.String())
		entry.Mimetype = mtype.String()
		_, err = s.sdr.SaveMedia(entry)
		if err != nil {
			return err
		}
		fob.Close()
	}
	return err
}

func (s service) ExtractCreationDate(force bool) error {
	mtFilter := []string{"image/jpeg", "video/mp4", "image/png"}
	//mtFilter := []string{"video/mp4", "image/png"}
	medialist, err := s.sdr.GetFilesByMimetypeFilter(mtFilter)
	if err != nil {
		return err
	}
	for i := range medialist {
		dt, err := s.ExtractExifDataFromFile(medialist[i])
		if err != nil {
			dt, err = s.ExtractDateByFilename(medialist[i])
		}
		if err == nil {
			medialist[i].CreationDate = dt
			_, err = s.sdr.SaveMedia(medialist[i])
			if err != nil {
				return err
			}
		} else {
			log.Printf("could not find CreationDate for %v", medialist[i].Path)
		}
	}
	return nil
}

func (s service) ExtractExifDataFromFile(media *SourceMedia) (time.Time, error) {
	fob, err := os.Open(media.Path)
	defer func(fob *os.File) {
		err = fob.Close()
	}(fob)
	if err != nil {
		return time.Time{}, err
	}
	data, err := io.ReadAll(fob)
	if err != nil {
		return time.Time{}, err
	}
	rawExif, err := exif.SearchAndExtractExif(data)
	if err != nil {
		return time.Time{}, err
	}
	im, err := exifcommon.NewIfdMappingWithStandard()
	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return time.Time{}, err
	}
	tagName := "DateTime"
	rootIfd := index.RootIfd
	// We know the tag we want is on IFD0 (the first/root IFD).
	results, err := rootIfd.FindTagWithName(tagName)
	if err != nil {
		return time.Time{}, err
	}
	ite := results[0]
	valueRaw, err := ite.Value()
	if err != nil {
		return time.Time{}, err
	}
	value, err := time.Parse("2006:01:02 15:04:05", valueRaw.(string))
	return value, err
}

func (s service) ExtractDateByFilename(media *SourceMedia) (time.Time, error) {
	re, err := regexp.Compile(`([\d]{8})`)
	if err != nil {
		return time.Time{}, err
	}
	//bpath := []byte(media.Path)
	result := re.FindAllString(media.Path, -1)
	for j := range result {
		dt, err := time.Parse("20060102", result[j])
		if err == nil {
			return dt, nil
		}
	}
	return time.Time{}, err
}

func (s service) OrganizeToFolder(force bool) error {
	mtFilter := []string{"image/jpeg", "video/mp4", "image/png"}
	mtExt := []string{"jpg", "mp4", "png"}
	//mtFilter := []string{"video/mp4", "image/png"}
	err, sourceChecks := s.sdr.GetAllCheckSum()
	if err != nil {
		return err
	}
	for j := range sourceChecks {
		path := sourceChecks[j].Sources[0]
		media, err := s.sdr.GetFileByKey(path)
		if err != nil {
			return err
		}
		for i := range mtFilter {
			if mtFilter[i] == media.Mimetype {
				err = s.drf.ExportToDirectory(media, j, mtExt[i])
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}
