package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// PicSorter sorts pictures into a library, while extracting incoming duplicates, unsupported files, etc.
type PicSorter struct {
	isDryRun        bool
	deduper         *Deduper
	fileMover       *FileMover
	libDir          string
	duplicateDir    string
	trashedDir      string
	unsupportedDir  string
	matchLivePhotos bool // e.g. match video IMG_7299.MP4 as live photo to metadata from IMG_7299.HEIC.json
	local           *time.Location
}

// NewPicSorter creates a new PicSorter with the given Deduper and FileMover.
func NewPicSorter(isDryRun bool, deduper *Deduper, fileMover *FileMover, libDir string, duplicateDir string, trashedDir string, unsupportedDir string, matchLivePhotos bool) *PicSorter {
	result := new(PicSorter)
	result.isDryRun = isDryRun
	result.deduper = deduper
	result.fileMover = fileMover
	result.libDir = libDir
	result.duplicateDir = duplicateDir
	result.trashedDir = trashedDir
	result.unsupportedDir = unsupportedDir
	result.matchLivePhotos = matchLivePhotos
	// Workaround to get "local" location. "Time.Local()" does not pick the right offset for DST state.
	zoneName, offset := time.Now().Zone()
	result.local = time.FixedZone(zoneName, offset)
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
			if isUnsupportedFileByExtension(path) {
				log.Println("[INFO] Treating file as 'unsupported' based on its extension", path)
				unsupportedPaths = append(unsupportedPaths, path)
				return nil
			}

			googleMetadata := sorter.getGooglePhotoMetadata(path)
			if googleMetadata != nil && googleMetadata.IsTrashed {
				log.Println("[INFO] Treating file as 'trashed' based on the metadata", path)
				err = sorter.handleTrashed(path, dirPath)
				if err != nil {
					log.Println("[WARN]", path, "Failed to check/handle trashed:", err)
				}
				return nil
			}

			newPath, err := sorter.deriveNewPathFromFileMetadata(path)
			if err != nil {
				if googleMetadata != nil {
					newPath, err = sorter.deriveNewPathFromGoogleMetadata(path, googleMetadata)
				}
				if err != nil {
					// The file is unsupported.  Nevertheless, check for duplicates.
					// This is realy only useful with eager deduping, but it could save us from having to care about why the file is unsupported.
					isDuplicate, err := sorter.checkAndHandleIndexedDupes(path, dirPath)
					if err != nil {
						log.Println("[WARN]", path, "Failed to check/handle indexed duplicates:", err)
					} else if isDuplicate {
						log.Println("[INFO] Treating file as 'duplicate' (unsupported)", path)
					} else {
						log.Println("[INFO] Treating file as 'unsupported' due to lack of metadata (file or Google)", path)
						unsupportedPaths = append(unsupportedPaths, path)
					}
					return nil
				}
			}

			isDuplicate, err := sorter.checkAndHandleDupes(path, dirPath, newPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to check/handle duplicates:", err)
				return nil
			} else if isDuplicate {
				log.Println("[INFO] Treating file as 'duplicate'", path)
				return nil
			}

			log.Println("[INFO] Relocating file", path)
			destPath, err := sorter.fileMover.MoveFileWithRename(path, newPath)
			if err != nil {
				log.Println("[WARN]", path, "Failed to sort file:", err)
				return nil
			}

			err = sorter.deduper.AddFileToIndex(destPath)
			if err != nil {
				if !sorter.isDryRun {
					log.Println("[WARN]", path, "Failed to index file:", err)
				}
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

func (sorter PicSorter) getGooglePhotoMetadata(filePath string) *GooglePhotoMetadata {
	metadata, _, err := NewGooglePhotoMetadata(filePath, sorter.matchLivePhotos)
	if err != nil {
		// probably no metadata
		return nil
	}
	return metadata
}

func (sorter PicSorter) handleTrashed(filePath string, fileRoot string) error {
	_, err := sorter.fileMover.MoveFileWithPreservedPath(filePath, fileRoot, sorter.trashedDir)
	if err != nil {
		return err
	}
	return nil
}

func (sorter PicSorter) checkAndHandleDupes(filePath string, fileRoot string, newPath string) (bool, error) {
	newPathDir := filepath.Dir(newPath)
	err := sorter.deduper.AddDirectoryToIndex(newPathDir)
	if err != nil {
		return false, err
	}
	return sorter.checkAndHandleIndexedDupes(filePath, fileRoot)
}

func (sorter PicSorter) checkAndHandleIndexedDupes(filePath string, fileRoot string) (bool, error) {
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

	result := sorter.deriveNewPathFromTimestamp(filePath, timestamp)
	return result, nil
}

func (sorter PicSorter) deriveNewPathFromGoogleMetadata(filePath string, metadata *GooglePhotoMetadata) (string, error) {
	if metadata.PhotoTakenTime.IsZero() {
		return "", errors.New("no 'PhotoTakenTime' present in Google metadata")
	}

	result := sorter.deriveNewPathFromTimestamp(filePath, metadata.PhotoTakenTime)
	return result, nil
}

func (sorter PicSorter) deriveNewPathFromTimestamp(filePath string, timestamp time.Time) string {
	localTimestamp := timestamp.In(sorter.local)
	filename := filepath.Base(filePath)
	yeardir := localTimestamp.Format("2006")
	datedir := localTimestamp.Format("2006-01-02")
	fileprefix := localTimestamp.Format("2006-01-02_15-04-05_")

	result := filepath.Join(sorter.libDir, yeardir, datedir, fileprefix+filename)
	log.Println("[DEBUG] Derived path", result, "from timestamp", timestamp.String(), "localized to", localTimestamp.String())

	return result
}

func isUnsupportedFileByExtension(picFilePath string) bool {
	ext := filepath.Ext(picFilePath)
	regex := regexp.MustCompile(`\.[jJ][sS][oO][nN]$`) // .json .JSON .Json ...
	matches := regex.Find([]byte(ext))

	return (matches != nil)
}
