package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FileMover moves files, with capability of "dry run".
type FileMover struct {
	isDryRun           bool
	undoScriptFilePath string
}

// NewFileMover creates a new FileMover with given dryrun state.
func NewFileMover(isDryRun bool, undoScriptFilePath string) *FileMover {
	result := new(FileMover)
	result.isDryRun = isDryRun
	result.undoScriptFilePath = undoScriptFilePath
	return result
}

// MoveFileWithRename moves the specified file to the specified path creating the directory if needed, and renaming the file if needed to avoid collission.
func (fileMover FileMover) MoveFileWithRename(sourcePath string, destPath string) (string, error) {
	if !fileMover.isDryRun {
		destDir := filepath.Dir(destPath)
		if err := fileMover.writeUndoCommandForDirCreate(destDir); err != nil {
			return "", err
		}
		if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
			return "", err
		}
		destPath, err := getNonCollidingPath(destPath)
		if err != nil {
			return "", err
		}
		log.Println("[INFO]", "Moving file", sourcePath, "to", destPath)
		if err := fileMover.writeUndoCommandForFileMove(sourcePath, destPath); err != nil {
			return "", err
		}
		// I frequently get "invalid cross-device link" with os.Rename even on same partition.
		//if err := os.Rename(sourcePath, destPath); err != nil {
		if err := fileMover.moveFile(sourcePath, destPath); err != nil {
			return "", err
		}
	} else {
		log.Println("[INFO]", "Dryrun moving file", sourcePath, "to", destPath)
	}
	return destPath, nil
}

// MoveFileWithPreservedPath moves the specified source file (which must have the given root) to the specified destination root.  Returns the destination path.
func (fileMover FileMover) MoveFileWithPreservedPath(sourcePath string, sourceRoot string, destRoot string) (string, error) {
	relPath, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil {
		return "", err
	}
	destPath := filepath.Join(destRoot, relPath)
	if _, err = fileMover.MoveFileWithRename(sourcePath, destPath); err != nil {
		return "", err
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
		if err := fileMover.writeUndoCommandForDirDelete(dirPath); err != nil {
			return err
		}
		os.Remove(dirPath)
	} else {
		log.Println("[INFO]", "Dryrun deleting directory", dirPath)
	}
	return nil
}

func (fileMover FileMover) moveFile(sourceFilePath string, destFilePath string) error {
	_, err := exec.Command("rsync", "-a", "--remove-source-files", sourceFilePath, destFilePath).Output()
	return err
}

func (fileMover FileMover) writeUndoCommandForFileMove(sourceFilePath string, destFilePath string) error {
	return fileMover.writeUndoCommand("rsync -avh --progress --remove-source-files \"" + destFilePath + "\" \"" + sourceFilePath + "\"")
}

func (fileMover FileMover) writeUndoCommandForDirDelete(dirPath string) error {
	return fileMover.writeUndoCommand("mkdir \"" + dirPath + "\"")
}

func (fileMover FileMover) writeUndoCommandForDirCreate(dirPath string) error {
	return fileMover.writeUndoCommand("rmdir \"" + dirPath + "\"")
}

func (fileMover FileMover) writeUndoCommand(undoCommand string) error {
	f, err := os.OpenFile(fileMover.undoScriptFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0744)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(undoCommand + "\n"); err != nil {
		return err
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
		return "", errors.New("failed to de-collide in " + strconv.Itoa(i) + " tries")
	}
	return resultPath, nil
}
