package imports

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"io/fs"
	"log"
	"os"
)

type SourceFileRepository interface {
	GetSourceFiles(func(path string, info fs.DirEntry, err error) error) error
	GetSourceFile(fpath string) (*os.File, error)
}

type Service interface {
	ScanSourceDirectory() error
	DetectMimetype(force bool) error
	ComputeChecksums(force bool) error
	ExtractExifData(force bool) error
}

type SourceDbRepository interface {
	AddFile(filename string) (string, error)
	HasFile(fileName string) (bool, error)
	GetFilesByMimetypeFilter(filter []string) ([]*SourceMedia, error)
	GetAllFiles() ([]*SourceMedia, error)
	SaveMedia(media *SourceMedia) (string, error)
	AddChecksum(media *SourceMedia) error
}

type service struct {
	sfr SourceFileRepository
	sdr SourceDbRepository
}

func NewService(sfr SourceFileRepository, sdr SourceDbRepository) Service {
	return &service{
		sfr: sfr,
		sdr: sdr,
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
		defer fob.Close()
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
		defer fob.Close()
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

	}
	return err
}

func (s service) ExtractExifData(force bool) error {
	mtFilter := []string{"image/jpeg", "video/mp4", "image/png"}
	medialist, err := s.sdr.GetFilesByMimetypeFilter(mtFilter)
	for i := range medialist {
		log.Printf("%v", medialist[i])
	}
	if err != nil {
		return err
	}

	return err
}
