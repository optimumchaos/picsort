package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strconv"
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

			// TODO
			// detect & skip files that seem to be duplicated, e.g. "foo.JPG(1).json", just in case.

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
	// TODO: This is not safe.  I found this situation:
	// /folder/fileA.jpg
	// /folder/fileA.JPG.json
	// /folder/fileA(1).jpg
	// /folder/fileA.JPG(1).json
	// You would expect "fileA.jpg" to match "fileA.JPG.json", and this change will allow that.
	// This change does not address the (1) problem.  I was going to deal with that later...
	// But this is not the case.  I found an instance where "fileA.jpg" corresponds to "fileA.JPG(1).json".
	// Google did not maintain order while exporting the files to disk.
	// I avoided a bad sort in this case only because of the filename extension case difference.  I used the file's metadata.
	// If I'd had this case fix (commented out below), I would have preferred the metadata and exported with the wrong date.
	// Removing this for now.  Ideas:
	// 1. prefer embedded dates when present
	// 2. only use metadata if the embedded dates and metadata agree.  (This would be a problem when the file has no metadata.)
	// 3. only use metadata if there is no (#) situation, implying duplicate files.
	// TODO!!!

	// ext := filepath.Ext(picFilePath)
	// upperExt := strings.ToUpper(ext)
	// lowerExt := strings.ToLower(ext)
	// noExtension := strings.TrimSuffix(picFilePath, ext)

	possibilities := []string{
		//		noExtension + upperExt + ".json",
		//		noExtension + lowerExt + ".json",
		picFilePath + ".json",
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
