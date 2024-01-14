package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// FileIndex indexes files by hash and path for directories.
type FileIndex struct {
	hashToPath        map[string]string
	hashedDirectories map[string]bool
}

// NewFileIndex creates a default instance of FileIndex.
func NewFileIndex() *FileIndex {
	result := new(FileIndex)
	result.hashToPath = make(map[string]string)
	result.hashedDirectories = make(map[string]bool)
	return result
}

// IsDirectoryIndexed determines whether the specified directory has been indexed by BuildIndexForDirectory.
func (fileIndex FileIndex) IsDirectoryIndexed(dirPath string) bool {
	_, isPresent := fileIndex.hashedDirectories[dirPath]
	return isPresent
}

// BuildIndexForDirectory recursively scans the given directory and indexes all the files contained within.
func (fileIndex FileIndex) BuildIndexForDirectory(dirPath string) error {
	log.Println("[DEBUG]", "Building index for path:", dirPath)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			err := fileIndex.AddFileToIndex(path)
			if err != nil {
				log.Println("[WARN]", "Skipping", path, ":", err)
			}
		} else {
			fileIndex.hashedDirectories[path] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	fileIndex.hashedDirectories[dirPath] = true
	return nil
}

// AddFileToIndex adds the specified file to the index.
func (fileIndex FileIndex) AddFileToIndex(filePath string) error {
	hash, err := deriveHashFromFile(filePath)
	if err != nil {
		return err
	}
	log.Println("[DEBUG]", "Adding to index:", hash, filePath)
	fileIndex.hashToPath[hash] = filePath
	return nil
}

// IsFilePresent determines whether the specified file has been indexed.
func (fileIndex FileIndex) IsFilePresent(filePath string) (bool, error) {
	hash, err := deriveHashFromFile(filePath)
	if err != nil {
		return false, err
	}
	_, isPresent := fileIndex.hashToPath[hash]
	return isPresent, nil
}

func deriveHashFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	fileSizeExtension := ""
	fileInfo, err2 := file.Stat()
	if err2 != nil {
		log.Println("[WARN]", "Unable to read file size:", err)
	} else {
		fileSizeExtension = "-" + strconv.FormatInt(fileInfo.Size(), 10)
	}

	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	result := hex.EncodeToString(hashInBytes) + fileSizeExtension
	return result, nil
}
