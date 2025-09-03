package domain

import (
	"io"

	"github.com/Rhaqim/buckt/internal/model"
)

type FolderService interface {
	CreateFolder(user_id, parent_id, folder_name, description string) (string, error)
	GetRootFolder(user_id string) (*model.FolderModel, error)
	GetFolder(user_id, folder_id string) (*model.FolderModel, error)
	GetFolders(parent_id string) ([]model.FolderModel, error)
	MoveFolder(folder_id, new_parent_id string) error
	RenameFolder(user_id, folder_id, new_name string) error
	DeleteFolder(folder_id string) (string, error)
	ScrubFolder(user_id, folder_id string) (string, error)
}

type FileService interface {
	CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error)
	GetFile(file_id string) (*model.FileModel, error)
	GetFileStream(file_id string) (*model.FileModel, io.ReadCloser, error)
	GetFiles(parent_id string) ([]model.FileModel, error)
	MoveFile(file_id, new_parent_id string) error
	RenameFile(file_id, new_name string) error
	UpdateFile(user_id, file_id, new_file_name string, new_file_data []byte) error
	DeleteFile(file_id string) (string, error)
	ScrubFile(file_id string) (string, error)
}
