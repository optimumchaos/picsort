package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
)

const flagDedupeDestDir = "DESTDIR"
const flagDedupeAll = "ALL"

func main() {
	libDirPtr := flag.String("libdir", "", "The directory containing your photo library.")
	incomingDirPtr := flag.String("incomingdir", "", "The directory with incoming photos.")
	dedupeDestDirPtr := flag.String("dedupedestdir", "", "The directory to which duplicate images will be moved.  The directory is created if it does not exist.")
	dedupePtr := flag.String("dedupe", flagDedupeDestDir, flagDedupeDestDir+" = dedupe against folders in destination directory. "+flagDedupeAll+" = dedupe against all files in the library.")
	isDryrunPtr := flag.Bool("dryrun", false, "Whether to do a dry run.")
	flag.Parse()
	if len(*libDirPtr) <= 0 ||
		len(*incomingDirPtr) <= 0 ||
		len(*dedupeDestDirPtr) <= 0 ||
		(*dedupePtr != flagDedupeAll && *dedupePtr != flagDedupeDestDir) {
		flag.Usage()
		os.Exit(2)
	}

	fmt.Println("Sorting incoming pictures from", *incomingDirPtr, "into library", *libDirPtr, ".")
	if *dedupePtr == flagDedupeDestDir {
		fmt.Println("Deduplicating only against the destination directory.")
	} else {
		fmt.Println("Deduplicating against the complete library.")
	}
	fmt.Println("Moving duplicates to", *dedupeDestDirPtr, ".")
	if *isDryrunPtr {
		fmt.Println("Dry run only.")
	}

	// If deduping all, make a hash index of the whole library

	// Walk the incoming. For each:
	//   Get the date/time & derive a new path
	//   If not deduping all, make a hash index of the new path directory
	//   If the hash index is not empty, dedupe
	//   If not duplicate, move to the new path

	// Dedupe
	//   Hash the incoming file
	//   If present in the index, move the incoming file to the dedupe directory, with full original path.

	// Move
	//   If the new file is already present, append a number starting with 1 to the new name.  Repeat.
	//   Move the file to the new name.
	//   Hash the file and add it (with new path) to the index.

	err := filepath.Walk(*incomingDirPtr, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// hash, err := deriveHashFromFile(path)
			// if err != nil {
			// 	fmt.Println(path, "Failed to get hash:", err)
			// } else {
			// 	fmt.Println(path, hash)
			// }
			newpath, err := getNewPathFromFile(path)
			if err != nil {
				fmt.Println(path, "Failed to get metadata:", err)
			} else {
				fmt.Println(path, newpath)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Failed:", err)
	}
}

func deriveHashFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	result := hex.EncodeToString(hashInBytes)
	return result, nil
}

func getNewPathFromFile(filePath string) (string, error) {
	picFile, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	metadata, err := exif.Decode(picFile)
	if err != nil {
		return "", err
	}

	timestamp, err := metadata.DateTime()
	if err != nil {
		return "", err
	}

	filename := filepath.Base(filePath)
	yeardir := timestamp.Format("2006")
	datedir := timestamp.Format("2006-01-02")
	fileprefix := timestamp.Format("2006-01-02_15-04-05_")

	result := filepath.Join(yeardir, datedir, fileprefix+filename)
	return result, nil
}
