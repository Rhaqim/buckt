package domain

type StorageFileService interface {
	UploadFile(file []byte, filename string) error
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
}
