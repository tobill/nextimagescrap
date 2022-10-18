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
	defer s.CloseDb()

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
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer s.CloseDb()

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}
	importService := imports.NewService(fs, s)
	err = importService.ComputeChecksums(false)

}
func actionExtractExifData(sourcePath *string) {
	log.Printf(*sourcePath)
	s, err := storage.NewSourceDbStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot open source db %v", err)
		os.Exit(0)
	}
	defer s.CloseDb()

	fs, err := storage.NewSourceFileStorage(*sourcePath)
	if err != nil {
		fmt.Printf("Cannot not find sourceapth %v", err)
		os.Exit(0)
	}
	importService := imports.NewService(fs, s)
	err = importService.ExtractExifData(false)

}

func main() {
	action := flag.String("action", "info", "action to do")
	sourcePath := flag.String("sourcePath", "", "source path of photos")
	flag.Parse()

	switch *action {
	case "info":
		fmt.Printf("heho print the info")
	case "scan-source":
		actionScanSourcepath(sourcePath)
	case "compute-checksum":
		actionComputeChecksum(sourcePath)
	case "extract-exif":
		actionExtractExifData(sourcePath)
	default:
		fmt.Printf("Nothing to do\n")
		fmt.Printf("Nothing to do\n")
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

}
