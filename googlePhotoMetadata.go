package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// GooglePhotoMetadata represents the metadata stored in a Google Photos JSON file.
type GooglePhotoMetadata struct {
	IsTrashed bool `json:"trashed"`
}

// NewGooglePhotoMetadata creates a new metadata instance from the given picture filename.  The convention is <picname>.json
func NewGooglePhotoMetadata(picFilePath string) (*GooglePhotoMetadata, string, error) {
	result := GooglePhotoMetadata{}
	metadataFilePath := picFilePath + ".json"
	file, err := ioutil.ReadFile(metadataFilePath)
	if err != nil {
		return nil, metadataFilePath, err
	}
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, metadataFilePath, err
	}
	log.Println("[DEBUG]", picFilePath, "IsTrashed:", result.IsTrashed)
	return &result, metadataFilePath, nil
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
