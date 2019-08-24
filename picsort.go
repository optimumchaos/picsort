package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const version = "0.02"

const flagDedupeLazy = "lazy"
const flagDedupeEager = "eager"

const dedupeSubDir = "duplicates"
const trashedSubDir = "trashed"
const unsupportedSubDir = "unsupported"

func main() {
	fmt.Println("picsort", version)

	libDir := flag.String("libdir", "", "The directory containing your photo library (destination for sort).")
	incomingDir := flag.String("incomingdir", "", "The directory with incoming photos (unsorted).")
	dedupe := flag.String("dedupe", flagDedupeLazy, "How to dedupe: "+flagDedupeLazy+" = dedupe lazily per destination directory, "+flagDedupeEager+" = dedupe eagerly across entire library.")
	rejectDir := flag.String("rejectdir", "", "The root directory to which rejected files will be moved.  Picsort will create subdirectories for duplicates, trashed, and files missing metadata.")
	isDryrun := flag.Bool("dryrun", false, "Whether to do a dry run.")
	flag.Parse()
	if len(*libDir) <= 0 ||
		len(*incomingDir) <= 0 ||
		len(*rejectDir) <= 0 ||
		(*dedupe != flagDedupeLazy && *dedupe != flagDedupeEager) {
		flag.Usage()
		os.Exit(2)
	}

	log.Println("[INFO]", "Sorting incoming pictures from", *incomingDir, "into library", *libDir)
	log.Println("[INFO]", "Deduping set to", *dedupe)
	log.Println("[INFO]", "Moving rejects to", *rejectDir)
	if *isDryrun {
		log.Println("[INFO]", "Dry run only")
	}

	dedupeDir := filepath.Join(*rejectDir, dedupeSubDir)
	trashedDir := filepath.Join(*rejectDir, trashedSubDir)
	unsupportedDir := filepath.Join(*rejectDir, unsupportedSubDir)

	fileMover := NewFileMover(*isDryrun)
	fileIndex := NewFileIndex()
	deduper := NewDeduper(fileIndex, dedupeDir, *incomingDir, fileMover)
	sorter := NewPicSorter(deduper, fileMover, *libDir, dedupeDir, trashedDir, unsupportedDir)

	if *dedupe == flagDedupeEager {
		fileIndex.BuildIndexForDirectory(*libDir)
	}

	err := sorter.Sort(*incomingDir)
	if err != nil {
		log.Fatalln("[FATAL]", "Failed to sort incoming pictures in", *incomingDir, ":", err)
	}
}
