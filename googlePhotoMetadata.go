package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
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
func NewGooglePhotoMetadata(picFilePath string) (*GooglePhotoMetadata, string, error) {
	result := GooglePhotoMetadata{}
	metadataFilePaths := getMetadataFilenames(picFilePath)
	var err error
	var metadataFilePath string
	var file []byte
	for _, metadataFilePath = range metadataFilePaths {
		file, err = ioutil.ReadFile(metadataFilePath)
		if err == nil {
			err = json.Unmarshal(file, &result)
			if err != nil {
				return nil, metadataFilePath, err
			}
			log.Println("[DEBUG] Using metadata file:", metadataFilePath)
			log.Println("[DEBUG]", picFilePath, "IsTrashed:", result.IsTrashed)
			return &result, metadataFilePath, nil
		}
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

func getMetadataFilenames(picFilePath string) []string {
	ext := filepath.Ext(picFilePath)
	upperExt := strings.ToUpper(ext)
	lowerExt := strings.ToLower(ext)
	noExtension := strings.TrimSuffix(picFilePath, ext)

	possibilities := []string{
		noExtension + upperExt + ".json",
		noExtension + lowerExt + ".json",
	}
	return possibilities
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
