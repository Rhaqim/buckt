package domain

import (
	"mime/multipart"

	"github.com/Rhaqim/buckt/request"
)

type ManagerService interface {
	CreateOwner(name, email string) error
	CreateBucket(name, description, ownerID string) error
	DeleteBucket(bucketName string) error
	GetBuckets(ownerID string) ([]interface{}, error)
}

type FileService interface {
	UploadFile(file *multipart.FileHeader, bucketName string, folderPath string) error
	DownloadFile(request.FileRequest) ([]byte, error)
	RenameFile(request.RenameFileRequest) error
	MoveFile(request.MoveFileRequest) error
	ServeFile(request.FileRequest, bool) (string, error)
	DeleteFile(request.FileRequest) error
}

type FolderService interface {
	CreateFolder(bucketName, folderPath string) error
	RenameFolder(request.RenameFolderRequest) error
	MoveFolder(request.MoveFolderRequest) error
	DeleteFolder(request.BaseFileRequest) error
	GetFilesInFolder(request.BaseFileRequest) ([]interface{}, error)
	GetSubFolders(request.BaseFileRequest) ([]interface{}, error)
}

type BucktService interface {
	ManagerService
	FileService
	FolderService
}
