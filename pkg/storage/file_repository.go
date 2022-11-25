package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"nextimagescrap/pkg/imports"
	"os"
	"path/filepath"
)

type SourceFileStorage struct {
	sourcePath string
}

// NewFileStorage create new file storage object
func NewSourceFileStorage(sourcePath string) (*SourceFileStorage, error) {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, err
	}

	s := SourceFileStorage{
		sourcePath: sourcePath,
	}

	return &s, nil
}
func (s SourceFileStorage) GetSourceFiles(walkFunc func(path string, info fs.DirEntry, err error) error) error {
	err := filepath.WalkDir(s.sourcePath, walkFunc)
	return err
}

type DestinationFileStorage struct {
	destinationPath string
}

func (d *DestinationFileStorage) getTargetPath(media *imports.SourceMedia) (string, error) {
	datepath := filepath.Join(d.destinationPath, media.CreationDate.Format("2006"),
		media.CreationDate.Format("01"))
	if _, err := os.Stat(datepath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(datepath, os.ModePerm); err != nil {
			return "", err
		}
	}
	return datepath, nil
}

func (d *DestinationFileStorage) ExportToDirectory(media *imports.SourceMedia, i int, ext string) error {
	log.Printf("exporting %v", media)
	targetPath, err := d.getTargetPath(media)
	if err != nil {
		return err
	}
	basefilename := fmt.Sprintf("image_%s_%d.%s", media.CreationDate.Format("20060102"), i, ext)
	destFilename := filepath.Join(targetPath, basefilename)
	_, err = copyFile(media.Path, destFilename)
	return err
}

// GetSourceFile returns origina file by path
func (s *SourceFileStorage) GetSourceFile(fpath string) (*os.File, error) {
	return os.Open(fpath)
}

// NewFileStorage create new file storage object
func NewDestinationFileStorage(destPath string) (*DestinationFileStorage, error) {
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return nil, err
	}

	s := DestinationFileStorage{
		destinationPath: destPath,
	}
	return &s, nil
}

func copyFile(src string, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func(source *os.File) {
		err = source.Close()
	}(source)

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer func(destination *os.File) {
		err = destination.Close()
	}(destination)

	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
