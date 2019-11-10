package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const version = "0.04"

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
	undoScriptFilePath := flag.String("undofile", "undo.sh", "The name of a file in which to write undo commands.")
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

	tempUndoScriptFilePath := *undoScriptFilePath + ".temp"
	dedupeDir := filepath.Join(*rejectDir, dedupeSubDir)
	trashedDir := filepath.Join(*rejectDir, trashedSubDir)
	unsupportedDir := filepath.Join(*rejectDir, unsupportedSubDir)

	fileMover := NewFileMover(*isDryrun, tempUndoScriptFilePath)
	fileIndex := NewFileIndex()
	deduper := NewDeduper(fileIndex, dedupeDir, *incomingDir, fileMover)
	sorter := NewPicSorter(deduper, fileMover, *libDir, dedupeDir, trashedDir, unsupportedDir)

	if *dedupe == flagDedupeEager {
		fileIndex.BuildIndexForDirectory(*libDir)
	}

	if !*isDryrun {
		initializeUndoFile(tempUndoScriptFilePath, *undoScriptFilePath)
	}
	sortErr := sorter.Sort(*incomingDir)
	var undoFileErr error
	if !*isDryrun {
		undoFileErr = writeUndoFile(tempUndoScriptFilePath, *undoScriptFilePath)
	}
	if sortErr != nil {
		log.Fatalln("[FATAL]", "Failed to sort incoming pictures in", *incomingDir, ":", sortErr)
	}
	if !*isDryrun {
		if undoFileErr != nil {
			log.Fatalln("[FATAL]", "Failed to write undo file:", undoFileErr)
		} else {
			log.Println("[INFO]", "To reinstate rejected files, execute", *undoScriptFilePath)
		}
	}
}

func initializeUndoFile(tempUndoFilePath string, undoFilePath string) {
	os.Remove(tempUndoFilePath)
	os.Chmod(undoFilePath, 0644)
	os.Rename(undoFilePath, undoFilePath+"."+time.Now().Format(time.RFC3339))
}

func writeUndoFile(tempUndoFilePath string, undoFilePath string) error {
	tempFile, err := os.Open(tempUndoFilePath)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	var lines []string
	scanner := bufio.NewScanner(tempFile)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	file, err := os.OpenFile(undoFilePath, os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "#!/bin/sh")
	lineCount := len(lines)
	for i := lineCount - 1; i >= 0; i-- {
		fmt.Fprintln(w, lines[i])
	}
	if err := w.Flush(); err != nil {
		return err
	}

	tempFile.Close()
	os.Remove(tempUndoFilePath)

	return nil
}
