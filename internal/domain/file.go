package domain

type StorageFileService interface {
	CreateOwner(name, email string) error
	CreateBucket(name, description, ownerID string) error
	UploadFile(file []byte, bucketname, filename string) error
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
	GetFiles(bucketName string) ([]interface{}, error)
	Serve(filename string, serve bool) (string, error)
}
