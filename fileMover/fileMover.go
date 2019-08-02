package fileMover

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FileMover moves files, with capability of "dry run".
type FileMover struct {
	isDryRun bool
}

// New creates a new FileMover with given dryrun state.
func New(isDryRun bool) *FileMover {
	result := new(FileMover)
	result.isDryRun = isDryRun
	return result
}

// MoveFileWithRename moves the specified file to the specified path creating the directory if needed, and renaming the file if needed to avoid collission.
func (fileMover FileMover) MoveFileWithRename(sourcePath string, destPath string) (string, error) {
	if !fileMover.isDryRun {
		err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
		if err != nil {
			return "", err
		}
		destPath, err := getNonCollidingPath(destPath)
		log.Println("[INFO]", "Moving file", sourcePath, "to", destPath)
		if err != nil {
			return "", err
		}
		err = os.Rename(sourcePath, destPath)
		if err != nil {
			return "", err
		}
	} else {
		log.Println("[INFO]", "Dryrun moving file", sourcePath, "to", destPath)
	}
	return destPath, nil
}

// DeleteEmptyDirectories deletes any empty directories that can be deleted, rooted at the specified directory.
func (fileMover FileMover) DeleteEmptyDirectories(dirPath string) error {
	if !fileMover.isDryRun {
		fileNames, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return err
		}
		for _, entry := range fileNames {
			if entry.IsDir() {
				childDirPath := filepath.Join(dirPath, entry.Name())
				err := fileMover.DeleteEmptyDirectories(childDirPath)
				if err != nil {
					log.Println("[WARN]", "Failed to delete directory", childDirPath, ":", err)
				}
			}
		}
		log.Println("[INFO]", "Attempting to delete directory ", dirPath)
		os.Remove(dirPath)
	} else {
		log.Println("[INFO]", "Dryrun deleting directory", dirPath)
	}
	return nil
}

func getNonCollidingPath(path string) (string, error) {
	var i int
	var resultPath = path
	var _, err = os.Stat(resultPath)
	for i = 1; !os.IsNotExist(err) && i <= 10; i++ {
		fileNameWithoutExtension := strings.Trim(filepath.Base(path), filepath.Ext(path))
		revisedFileName := strings.Join([]string{fileNameWithoutExtension, strconv.Itoa(i) + filepath.Ext(path)}, ".")
		resultPath = filepath.Join(filepath.Dir(path), revisedFileName)
		_, err = os.Stat(resultPath)
	}
	if i > 10 {
		return "", errors.New("Failed to de-collide in " + strconv.Itoa(i) + " tries.")
	}
	return resultPath, nil
}
