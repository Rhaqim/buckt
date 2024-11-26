package domain

import (
	"mime/multipart"

	"github.com/Rhaqim/buckt/request"
	"github.com/google/uuid"
)

type ManagerService interface {
	CreateOwner(name, email string) error
	CreateBucket(name, description, ownerID string) error
	DeleteBucket(bucketName string) error
	GetBucket(bucketName string) (interface{}, error)
	GetBuckets(ownerID uuid.UUID) ([]interface{}, error)
}

type FileService interface {
	UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error
	DownloadFile(request.FileRequest) ([]byte, error)
	RenameFile(request.RenameFileRequest) error
	MoveFile(request.MoveFileRequest) error
	ServeFile(string) (string, error)
	DeleteFile(request.FileRequest) error
}

type FolderService interface {
	CreateFolder(bucketName, folderPath string) error
	RenameFolder(request.RenameFolderRequest) error
	MoveFolder(request.MoveFolderRequest) error
	DeleteFolder(request.BaseFileRequest) error
	GetFilesInFolder(request.BaseFileRequest) ([]interface{}, error)
	GetSubFolders(request.BaseFileRequest) ([]interface{}, error)
	GetDescendants(uuid.UUID) ([]interface{}, error)
}

type BucktService interface {
	ManagerService
	FileService
	FolderService
}
