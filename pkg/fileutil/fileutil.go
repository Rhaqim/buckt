package fileutil

import (
	"fmt"
	"io"
	"mime/multipart"
)

// ProcessFile reads a multipart file header into memory and returns the filename and bytes.
func ProcessFile(file *multipart.FileHeader) (string, []byte, error) {
	return ProcessFileWithLimit(file, 0)
}

// ProcessFileWithLimit reads a multipart file with a size limit.
// If maxSize is 0, no limit is enforced.
func ProcessFileWithLimit(file *multipart.FileHeader, maxSize int64) (string, []byte, error) {
	// Reject early based on the Content-Length header
	if maxSize > 0 && file.Size > maxSize {
		return "", nil, fmt.Errorf("file size %d exceeds maximum allowed %d bytes", file.Size, maxSize)
	}

	f, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	var reader io.Reader = f
	if maxSize > 0 {
		reader = io.LimitReader(f, maxSize+1) // +1 to detect overflow
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, err
	}

	if maxSize > 0 && int64(len(data)) > maxSize {
		return "", nil, fmt.Errorf("file size exceeds maximum allowed %d bytes", maxSize)
	}

	return file.Filename, data, nil
}
