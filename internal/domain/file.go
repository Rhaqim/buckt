package domain

type StorageFileService interface {
	UploadFile(file []byte, bucketname, filename string) error
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
}
