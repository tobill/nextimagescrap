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
	ExportToDirectory(media *SourceMedia, ext string) error
}

type Service interface {
	ScanSourceDirectory() error
	DetectMimetype(force bool) error
	ComputeChecksums(force bool) error
	ExtractCreationDate(force bool) error
	OrganizeToFolder() error
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
		if err != nil {
			return err
		}
		h := sha1.New()
		_, err = io.Copy(h, fob)
		if err != nil {
			return err
		}
		entry.Checksum = hex.EncodeToString(h.Sum(nil))
		log.Printf("%s %s", entry.Path, entry.Checksum)
		_, err = s.sdr.SaveMedia(entry)
		err = s.sdr.AddChecksum(entry)
		if err != nil {
			return err
		}
		err = fob.Close()

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
		var mtype *mimetype.MIME
		if err == nil {
			mtype, err = mimetype.DetectReader(bytes.NewReader(b))
			log.Printf("%s %s", entry.Path, mtype.String())
		}
		if err != nil {
			log.Printf("could not detect mimetype for %v", entry.Path)
			entry.Mimetype = "unknown/error"
		} else {
			entry.Mimetype = mtype.String()
		}
		_, err = s.sdr.SaveMedia(entry)
		if err != nil {
			return err
		}
		err = fob.Close()
		if err != nil {
			return err
		}
	}
	return err
}

func (s service) ExtractCreationDate(force bool) error {
	mtFilter := []string{"image/jpeg", "image/png"}
	//mtFilter := []string{"video/mp4"}
	medialist, err := s.sdr.GetFilesByMimetypeFilter(mtFilter)
	if err != nil {
		return err
	}
	for i := range medialist {
		if medialist[i].CreationDate.Year() > 2000 && !force {
			//if medialist[i].Id != 19192 { // --C:\Data\Bilder\samples
			//log.Printf("CreationDate alr3eday present %v %v", medialist[i].Path, medialist[i].CreationDate)
			continue

		}
		log.Printf("search exif CreationDate for %v", medialist[i].Path)
		var dt time.Time
		if medialist[i].Mimetype != "video/mp4" {
			dt, err = s.ExtractExifDataFromFile(medialist[i])
		}
		if err != nil || medialist[i].Mimetype == "video/mp4" {
			log.Printf("%v", err)
			dt, err = s.ExtractDateByFilename(medialist[i])
		}
		if err == nil {
			medialist[i].CreationDate = dt
			log.Printf("found CreationDate for %v", medialist[i])
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
	defer fob.Close()
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
	rootIfd := index.RootIfd
	entries := rootIfd.DumpTags()
	createTime := time.Time{}
	for j := range entries {

		t := entries[j].TagName()
		log.Printf(t)
		if t == "DateTimeOriginal" || t == "DateTime" {
			v, _ := entries[j].Value()
			createTime, err = time.Parse("2006:01:02 15:04:05", v.(string))
		}
	}
	return createTime, err
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

func (s service) OrganizeToFolder() error {
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
				err = s.drf.ExportToDirectory(media, mtExt[i])
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}
