package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GooglePhotoMetadata represents the metadata stored in a Google Photos JSON file.
type GooglePhotoMetadata struct {
	IsTrashed      bool
	PhotoTakenTime time.Time
}

// NewGooglePhotoMetadata creates a new metadata instance from the given picture filename.  The convention is <picname>.json
func NewGooglePhotoMetadata(picFilePath string, matchLivePhotos bool) (*GooglePhotoMetadata, string, error) {
	log.Println("[DEBUG] Looking for metadata for", picFilePath)
	result := GooglePhotoMetadata{}
	metadataFilePaths := getMetadataFilenames(picFilePath, matchLivePhotos)
	var err error = nil
	var metadataFilePath string = "(no Google metadata file path found)"
	if isMetadataUnamgibuous(picFilePath, metadataFilePaths) {
		var file []byte
		for _, metadataFilePath = range metadataFilePaths {
			file, err = ioutil.ReadFile(metadataFilePath)
			if err == nil {
				err = json.Unmarshal(file, &result)
				if err != nil {
					log.Println("[WARN] Failed to unmarshal Google metadata file:", metadataFilePath, err)
					return nil, metadataFilePath, err
				}
				log.Println("[DEBUG] Using Google metadata file:", metadataFilePath)
				log.Println("[DEBUG]", picFilePath, "IsTrashed:", result.IsTrashed, "PhotoTakenTime:", result.PhotoTakenTime)

				return &result, metadataFilePath, nil
			}
		}
	} else {
		log.Println("[WARN] Skipping Google metadata for", picFilePath, "because it could not be matched to the file with full confidence.")
	}
	return nil, metadataFilePath, err
}

// UnmarshalJSON unmarshalls the Google Metadata format into the values of interest.
func (metadata *GooglePhotoMetadata) UnmarshalJSON(b []byte) error {

	var f interface{}
	json.Unmarshal(b, &f)

	allProps := f.(map[string]interface{})
	photoTakenTimeMap := allProps["photoTakenTime"]
	if photoTakenTimeMap != nil {
		photoTakenTimeProps := photoTakenTimeMap.(map[string]interface{})
		unixTimeString := photoTakenTimeProps["timestamp"].(string)
		if unixTimeString != "" {
			unixTimeInt64, err := strconv.ParseInt(unixTimeString, 10, 64)
			if err == nil {
				metadata.PhotoTakenTime = time.Unix(unixTimeInt64, 0)
			}
		}
	}

	isTrashed := allProps["trashed"]
	if isTrashed != nil {
		metadata.IsTrashed = isTrashed.(bool)
	}

	return nil
}

func getMetadataFilenames(picFilePath string, matchLivePhotos bool) []string {
	ext := filepath.Ext(picFilePath)
	upperExt := strings.ToUpper(ext)
	lowerExt := strings.ToLower(ext)
	noExtension := strings.TrimSuffix(picFilePath, ext)

	possibilities := []string{
		noExtension + upperExt + ".json",
		noExtension + lowerExt + ".json",
	}
	if matchLivePhotos {
		possibilities = append(possibilities,
			noExtension+".HEIC.json",
			noExtension+".heic.json",
		)
	}
	return possibilities
}

func isMetadataUnamgibuous(picFilePath string, filenames []string) bool {
	// Situation:
	// /folder/fileA.jpg
	// /folder/fileA.JPG.json
	// /folder/fileA(1).jpg
	// /folder/fileA.JPG(1).json
	// I found an instance where "fileA.jpg" corresponds to "fileA.JPG(1).json".
	// Google did not maintain order while exporting the files to disk.
	// Consider the metadata unambiguous if there is no indication of duplicate picfile or metadata filenames.

	log.Println("[DEBUG] Checking pic filename for duplicates:", picFilePath)
	if isFileDuplicated(picFilePath) {
		log.Println("[DEBUG] Treating metadata as ambiguous because picture file uses a potentially duplicated filename.")
		return false
	}
	for _, filename := range filenames {
		log.Println("[DEBUG] Checking potential metadata filename for duplicates:", filename)
		if isFileDuplicated(filename) {
			log.Println("[DEBUG] Treating metadata as ambiguous because metadata file uses a potentially duplicated filename.")
			return false
		}
	}
	return true
}

func isFileDuplicated(filePath string) bool {
	regex := regexp.MustCompile(`(\([\d]+\))?(` + filepath.Ext(filePath) + "$)") // foo(1).png or foo.png
	filePattern1 := string(regex.ReplaceAll([]byte(filePath), []byte("(*)$2")))  // foo(1).png -> foo(*).png
	fileCount1, err := countFiles(filePattern1)
	if err != nil {
		log.Println("[WARN] Assuming file duplication due to failure to list files for pattern:", filePattern1)
		return true
	}
	filePattern2 := string(regex.ReplaceAll([]byte(filePath), []byte("$2"))) // foo(1).png -> foo.png
	fileCount2, err := countFiles(filePattern2)
	if err != nil {
		log.Println("[WARN] Assuming file duplication due to failure to list files for pattern:", filePattern2)
		return true
	}
	return (fileCount1 + fileCount2) > 1
}

func countFiles(filePattern string) (int, error) {
	matches, err := filepath.Glob(filePattern)
	if err != nil {
		return 0, err
	}
	log.Println("[TRACE] Checked files for pattern", filePattern, matches)
	return len(matches), nil
}

// example: IMG_3560.JPG.json (latitude and longitude sanitized for the public)
// {
// 	"title": "IMG_3560.JPG",
// 	"description": "",
// 	"imageViews": "0",
// 	"creationTime": {
// 	  "timestamp": "1562771646",
// 	  "formatted": "Jul 10, 2019, 3:14:06 PM UTC"
// 	},
// 	"modificationTime": {
// 	  "timestamp": "1563340364",
// 	  "formatted": "Jul 17, 2019, 5:12:44 AM UTC"
// 	},
// 	"geoData": {
// 	  "latitude": <lat>,
// 	  "longitude": <lon>,
// 	  "altitude": 99.5951309328969,
// 	  "latitudeSpan": 0.0,
// 	  "longitudeSpan": 0.0
// 	},
// 	"geoDataExif": {
// 	  "latitude": <lat>,
// 	  "longitude": <lon>,
// 	  "altitude": 99.5951309328969,
// 	  "latitudeSpan": 0.0,
// 	  "longitudeSpan": 0.0
// 	},
// 	"photoTakenTime": {
// 	  "timestamp": "1562768659",
// 	  "formatted": "Jul 10, 2019, 2:24:19 PM UTC"
// 	},
// 	"trashed": true
//  }
