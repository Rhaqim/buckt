package domain

type StorageFileService interface {
	UploadFile(file []byte, bucketname, filename string) error
	Serve(filename string, serve bool) (string, error)
	DownloadFile(filename string) ([]byte, error)
	DeleteFile(filename string) error
	CreateBucket(name, description, ownerID string) error
	CreateOwner(name, email string) error
}
