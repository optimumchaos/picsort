package main

import (
	"flag"
	"log"
	"os"
)

const flagDedupeLazy = "lazy"
const flagDedupeEager = "eager"

func main() {
	libDir := flag.String("libdir", "", "The directory containing your photo library (destination for sort).")
	incomingDir := flag.String("incomingdir", "", "The directory with incoming photos (unsorted).")
	dedupe := flag.String("dedupe", flagDedupeLazy, "How to dedupe: "+flagDedupeLazy+" = dedupe lazily per destination directory, "+flagDedupeEager+" = dedupe eagerly across entire library.")
	dedupeDestDir := flag.String("dedupedestdir", "", "The directory to which duplicate images will be moved.  The directory is created if it does not exist.")
	isDryrun := flag.Bool("dryrun", false, "Whether to do a dry run.")
	flag.Parse()
	if len(*libDir) <= 0 ||
		len(*incomingDir) <= 0 ||
		len(*dedupeDestDir) <= 0 ||
		(*dedupe != flagDedupeLazy && *dedupe != flagDedupeEager) {
		flag.Usage()
		os.Exit(2)
	}

	log.Println("[INFO]", "Sorting incoming pictures from", *incomingDir, "into library", *libDir)
	log.Println("[INFO]", "Deduping set to", *dedupe)
	log.Println("[INFO]", "Moving duplicates to", *dedupeDestDir)
	if *isDryrun {
		log.Println("[INFO]", "Dry run only")
	}

	fileMover := NewFileMover(*isDryrun)
	fileIndex := NewFileIndex()
	deduper := NewDeduper(fileIndex, *dedupeDestDir, *incomingDir, fileMover)
	sorter := NewPicSorter(deduper, fileMover, *libDir)

	if *dedupe == flagDedupeEager {
		fileIndex.BuildIndexForDirectory(*libDir)
	}

	err := sorter.Sort(*incomingDir)
	if err != nil {
		log.Fatalln("[FATAL]", "Failed to sort incoming pictures in", *incomingDir, ":", err)
	}
}
