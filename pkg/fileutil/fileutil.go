package fileutil

import (
	"io"
	"mime/multipart"
)

// ProcessFile reads a multipart file header into memory and returns the filename and bytes.
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
