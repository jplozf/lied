package exifinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
)

// ExifData represents the structure of the JSON output from exiftool.
// We use map[string]interface{} to handle the dynamic nature of exiftool's output.
type ExifData map[string]interface{}

// ****************************************************************************
// GetExifInfo()
// ****************************************************************************
// GetExifInfo executes exiftool on the given file path and returns a map of key-value pairs.
func GetExifInfo(filePath string) (map[string]string, error) {
	cmd := exec.Command("exiftool", "-json", filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()
	/*
		if err != nil {
			return nil, fmt.Errorf("failed to run exiftool: %w, stderr: %s", err, stderr.String())
		}
	*/

	var exifOutput []ExifData
	if err := json.Unmarshal(stdout.Bytes(), &exifOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exiftool JSON output: %w, output: %s", err, stdout.String())
	}

	if len(exifOutput) == 0 {
		return nil, fmt.Errorf("no exif data found for file: %s", filePath)
	}

	// Convert ExifData to map[string]string for easier display
	result := make(map[string]string)
	for key, value := range exifOutput[0] { // exiftool -json returns an array with one object for a single file
		result[key] = fmt.Sprintf("%v", value)
	}

	return result, nil
}

// ****************************************************************************
// GetSortedExifInfo()
// ****************************************************************************
// GetSortedExifInfo returns exiftool information as a sorted slice of key-value pairs.
func GetSortedExifInfo(filePath string) ([]struct{ Key, Value string }, error) {
	exifMap, err := GetExifInfo(filePath)
	if err != nil {
		return nil, err
	}

	var sortedInfo []struct{ Key, Value string }
	keys := make([]string, 0, len(exifMap))
	for k := range exifMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sortedInfo = append(sortedInfo, struct{ Key, Value string }{Key: k, Value: exifMap[k]})
	}

	return sortedInfo, nil
}
