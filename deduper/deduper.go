package deduper

import (
	"path/filepath"

	"../fileIndex"
	"../fileMover"
)

// Deduper identifies and moves duplicate files.
type Deduper struct {
	fileIndex               *fileIndex.FileIndex
	duplicateDestinationDir string
	originalBaseDir         string
	duplicateFileMover      *fileMover.FileMover
}

// New creates a default instance of FileIndex.
func New(fileIndex *fileIndex.FileIndex, duplicateDestinationDir string, originalBaseDir string, duplicateFileMover *fileMover.FileMover) *Deduper {
	result := new(Deduper)
	result.fileIndex = fileIndex
	result.duplicateDestinationDir = duplicateDestinationDir
	result.originalBaseDir = originalBaseDir
	result.duplicateFileMover = duplicateFileMover
	return result
}

// AddDirectoryToIndex recursively indexes all files in the specified directory.
func (deduper Deduper) AddDirectoryToIndex(dirPath string) error {
	isIndexed := deduper.fileIndex.IsDirectoryIndexed(dirPath)
	if !isIndexed {
		deduper.fileIndex.BuildIndexForDirectory(dirPath)
		// Ignoring errors because the directory might not exist.
	}
	return nil
}

// AddFileToIndex indexe the specified file.
func (deduper Deduper) AddFileToIndex(filePath string) error {
	return deduper.fileIndex.AddFileToIndex(filePath)
}

// DedupeFile checks the index for duplicates of the specified file, and moves it to the dedupe dir if needed.
func (deduper Deduper) DedupeFile(filePath string) (bool, error) {
	isPresent, err := deduper.fileIndex.IsFilePresent(filePath)
	if err != nil {
		return false, err
	}
	if isPresent {
		relPath, err := filepath.Rel(deduper.originalBaseDir, filePath)
		if err != nil {
			return false, err
		}
		destPath := filepath.Join(deduper.duplicateDestinationDir, relPath)
		_, err = deduper.duplicateFileMover.MoveFileWithRename(filePath, destPath)
		if err != nil {
			return false, err
		}
	}
	return isPresent, nil
}
