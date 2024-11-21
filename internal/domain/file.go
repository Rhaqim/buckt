package domain

import "mime/multipart"

type StorageFileService interface {
	CreateOwner(name, email string) error
	CreateBucket(name, description, ownerID string) error
	UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
	GetFiles(bucketName string) ([]interface{}, error)
	Serve(filename string, serve bool) (string, error)
}
