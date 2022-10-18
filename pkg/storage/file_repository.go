package storage

import (
	"io/fs"
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

// GetSourceFile returns origina file by path
func (s *SourceFileStorage) GetSourceFile(fpath string) (*os.File, error) {
	return os.Open(fpath)
}

// NewFileStorage create new file storage object
func NewDestinationFileStorage(destPath string) *DestinationFileStorage {
	s := DestinationFileStorage{
		destinationPath: destPath,
	}
	return &s
}
