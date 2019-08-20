package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
)

// PicSorter sorts pictures into a library, while extracting incoming duplicates.
type PicSorter struct {
	deduper   *Deduper
	fileMover *FileMover
	libDir    string
}

// NewPicSorter creates a new PicSorter with the given Deduper and FileMover.
func NewPicSorter(deduper *Deduper, fileMover *FileMover, libDir string) *PicSorter {
	result := new(PicSorter)
	result.deduper = deduper
	result.fileMover = fileMover
	result.libDir = libDir
	return result
}

// Sort sorts pictures in the specified directory into the library.  Extracts duplicates to a special location.
func (sorter PicSorter) Sort(dirPath string) error {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			metadata, err := NewGooglePhotoMetadata(path)
			if err != nil {
				log.Println("[DEBUG]", "No recognizable Google Photos metadata for", path)
			} else if metadata.IsTrashed {
				log.Println("[DEBUG]", path, "Skipping: trashed")
				return nil
			}
			newPath, err := sorter.deriveNewPathFromFileMetadata(path)
			if err != nil {
				log.Println("[DEBUG]", path, "Skipping:", err)
				return nil
			}
			log.Println("[DEBUG]", "Sorting file", path, "to", newPath)
			newPathDir := filepath.Dir(newPath)
			err = sorter.deduper.AddDirectoryToIndex(newPathDir)
			if err != nil {
				log.Println("[WARN]", path, "Skipping:", err)
				return nil
			}
			isDuplicate, err := sorter.deduper.DedupeFile(path)
			if err != nil {
				log.Println("[WARN]", path, "Skipping:", err)
				return nil
			} else if isDuplicate {
				return nil
			}
			destPath, err := sorter.fileMover.MoveFileWithRename(path, newPath)
			if err != nil {
				log.Println("[WARN]", path, "Skipping:", err)
				return nil
			}
			err = sorter.deduper.AddFileToIndex(destPath)
			if err != nil {
				log.Println("[WARN]", path, "Skipping:", err)
				return nil
			}
		}
		return nil
	})
	sorter.fileMover.DeleteEmptyDirectories(dirPath)
	return err
}

func (sorter PicSorter) deriveNewPathFromFileMetadata(filePath string) (string, error) {
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

	result := filepath.Join(sorter.libDir, yeardir, datedir, fileprefix+filename)
	return result, nil
}
