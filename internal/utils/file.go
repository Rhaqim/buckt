package utils

import (
	"io"
	"mime/multipart"
	"strings"
	"unicode"
)

func ProcessFile(file *multipart.FileHeader) (string, []byte, error) {
	f, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", nil, err
	}

	return file.Filename, data, nil
}

// ValidateFolderPath validates a folder path and splits it into components if valid.
func ValidateFolderPath(folderPath string) []string {
	// Return an empty list if the folder path is empty
	if len(strings.TrimSpace(folderPath)) == 0 {
		return []string{}
	}

	// Trim spaces
	folderPath = strings.TrimSpace(folderPath)

	// Check for invalid characters
	if !isValidFolderPath(folderPath) {
		return []string{}
	}

	// Ensure no double slashes
	if strings.Contains(folderPath, "//") {
		return []string{}
	}

	// Remove leading and trailing slashes
	folderPath = strings.Trim(folderPath, "/")

	// Split the folder path into components
	return strings.Split(folderPath, "/")
}

// isValidFolderPath checks if a folder path contains only valid characters (alphanumeric, spaces, slashes).
func isValidFolderPath(s string) bool {
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == ' ' || r == '-') {
			return false
		}
	}
	return true
}
