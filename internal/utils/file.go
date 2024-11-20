package utils

import (
	"io"
	"mime/multipart"
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
