package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"
)

// PicSorter sorts pictures into a library, while extracting incoming duplicates, unsupported files, etc.
type PicSorter struct {
	deduper        *Deduper
	fileMover      *FileMover
	libDir         string
	duplicateDir   string
	trashedDir     string
	unsupportedDir string
}

// NewPicSorter creates a new PicSorter with the given Deduper and FileMover.
func NewPicSorter(deduper *Deduper, fileMover *FileMover, libDir string, duplicateDir string, trashedDir string, unsupportedDir string) *PicSorter {
	result := new(PicSorter)
	result.deduper = deduper
	result.fileMover = fileMover
	result.libDir = libDir
	result.duplicateDir = duplicateDir
	result.trashedDir = trashedDir
	result.unsupportedDir = unsupportedDir
	return result
}

// Sort sorts pictures in the specified directory into the library.  Extracts duplicates, trashed, and unsupported files to a special location.
func (sorter PicSorter) Sort(dirPath string) error {
	var unsupportedPaths []string
	log.Println("[INFO]", "Scanning incoming files from", dirPath)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			isTrashed, err := sorter.checkAndHandleTrashed(path, dirPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to check/handle trashed:", err)
				return nil
			} else if isTrashed {
				return nil
			}

			newPath, err := sorter.deriveNewPathFromFileMetadata(path)
			if err != nil {
				unsupportedPaths = append(unsupportedPaths, path)
				return nil
			}

			isDuplicate, err := sorter.checkAndHandleDupes(path, dirPath, newPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to check/handle duplicates:", err)
				return nil
			} else if isDuplicate {
				return nil
			}

			destPath, err := sorter.fileMover.MoveFileWithRename(path, newPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to sort file:", err)
				return nil
			}

			err = sorter.deduper.AddFileToIndex(destPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to index file:", err)
				return nil
			}
		}
		return nil
	})

	log.Println("[INFO]", "Cleaning up unsupported files.")
	for _, unsupportedPath := range unsupportedPaths {
		_, err := sorter.fileMover.MoveFileWithPreservedPath(unsupportedPath, dirPath, sorter.unsupportedDir)
		if err != nil {
			log.Println("[WARN]", unsupportedPath, "Failed to move unsupported file:", err)
		}
	}
	sorter.fileMover.DeleteEmptyDirectories(dirPath)

	return err
}

func (sorter PicSorter) checkAndHandleTrashed(filePath string, fileRoot string) (bool, error) {
	metadata, _, err := NewGooglePhotoMetadata(filePath)
	if err != nil {
		// probably no metadata
		return false, nil
	} else if metadata.IsTrashed {
		_, err := sorter.fileMover.MoveFileWithPreservedPath(filePath, fileRoot, sorter.trashedDir)
		if err != nil {
			return false, err
		}
		// This seems to cause problems for the file-walk:
		//sorter.fileMover.MoveFileWithPreservedPath(metadataFilePath, fileRoot, sorter.trashedDir)
	}
	return metadata.IsTrashed, nil
}

func (sorter PicSorter) checkAndHandleDupes(filePath string, fileRoot string, newPath string) (bool, error) {
	newPathDir := filepath.Dir(newPath)
	err := sorter.deduper.AddDirectoryToIndex(newPathDir)
	if err != nil {
		return false, err
	}
	isDuplicate, err := sorter.deduper.IsDuplicate(filePath)
	if err != nil {
		return false, err
	} else if isDuplicate {
		_, err := sorter.fileMover.MoveFileWithPreservedPath(filePath, fileRoot, sorter.duplicateDir)
		if err != nil {
			return true, err
		}
	}
	return isDuplicate, nil
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
