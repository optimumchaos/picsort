package main

// Deduper identifies and moves duplicate files.
type Deduper struct {
	fileIndex               *FileIndex
	duplicateDestinationDir string
	originalBaseDir         string
	duplicateFileMover      *FileMover
}

// NewDeduper creates a default instance of FileIndex.
func NewDeduper(fileIndex *FileIndex, duplicateDestinationDir string, originalBaseDir string, duplicateFileMover *FileMover) *Deduper {
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

// IsDuplicate determines whether the specified file is a duplicate.
func (deduper Deduper) IsDuplicate(filePath string) (bool, error) {
	isPresent, err := deduper.fileIndex.IsFilePresent(filePath)
	if err != nil {
		return false, err
	}
	return isPresent, nil
}
