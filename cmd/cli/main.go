package main

import (
	"flag"
	"fmt"
	"log"
	"nextimagescrap/pkg/imports"
	"nextimagescrap/pkg/storage"
	"os"
)

func actionScanSourcepath(sourcePath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer func(s *storage.DbSourceStorage) {
		err := s.CloseDb()
		if err != nil {
			fmt.Printf("Error closing db %v", err)
			os.Exit(0)
		}
	}(s)

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}
	importService := imports.NewService(fs, s)
	err = importService.ScanSourceDirectory()
	if err != nil {
		log.Printf("%v", err)
		os.Exit(5)
	}
	err = importService.DetectMimetype(false)
	if err != nil {
		log.Printf("%v", err)
		os.Exit(5)
	}

}

func actionComputeChecksum(sourcePath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		log.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer func(s *storage.DbSourceStorage) {
		err := s.CloseDb()
		if err != nil {
			log.Printf("cannot close source db %v", err)
			os.Exit(0)
		}
	}(s)

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		log.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}
	importService := imports.NewService(fs, s)
	err = importService.ComputeChecksums(false)
	log.Printf("error %v", err)
}

func listAll(sourcePath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer func(s *storage.DbSourceStorage) {
		err := s.CloseDb()
		if err != nil {
			log.Printf("cannot close source db %v", err)
			os.Exit(0)
		}
	}(s)
	me, err := s.GetAllFiles()
	if err != nil {
		fmt.Printf("Cannot get media entries %v", err)
		os.Exit(0)
	}
	for i := range me {
		log.Printf("Entry: %d: %v", i, me[i])
	}

}

func extractCreationDate(sourcePath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer func(s *storage.DbSourceStorage) {
		err := s.CloseDb()
		if err != nil {
			log.Printf("cannot close source db %v", err)
			os.Exit(0)
		}
	}(s)

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}

	importService := imports.NewService(fs, s)
	err = importService.ExtractCreationDate(false)
	if err != nil {
		log.Printf("%v", err)
		os.Exit(5)
	}

}

func reorganizeToFolder(sourcePath *string, destPath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer func(s *storage.DbSourceStorage) {
		err := s.CloseDb()
		if err != nil {
			log.Printf("cannot close source db %v", err)
			os.Exit(0)
		}
	}(s)

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}

	dfs, err := storage.NewDestinationFileStorage(*destPath)
	if err != nil {
		fmt.Printf("Cannot not find destpath %v", err)
		os.Exit(0)
	}

	organizeService := imports.NewOrganizeService(fs, s, dfs)
	err = organizeService.OrganizeToFolder()
	if err != nil {
		log.Printf("Error reornanize: %v", err)
		os.Exit(0)
		return
	}
}

func main() {
	action := flag.String("action", "info", "action to do")
	sourcePath := flag.String("sourcePath", "", "source path of photos")
	destPath := flag.String("destPath", "", "dest path of photos")
	flag.Parse()

	switch *action {
	case "info":
		listAll(sourcePath)
	case "scan-source":
		actionScanSourcepath(sourcePath)
	case "compute-checksum":
		actionComputeChecksum(sourcePath)
	case "extract-creationdate":
		extractCreationDate(sourcePath)
	case "reorganize":
		reorganizeToFolder(sourcePath, destPath)
	default:
		fmt.Printf("Nothing to do\n")
		fmt.Printf("Nothing to do\n")
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

}
