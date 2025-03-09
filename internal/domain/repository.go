package domain

import (
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type FolderRepository interface {
	Create(folder *model.FolderModel) (string, error)
	GetFolder(folder_id uuid.UUID) (*model.FolderModel, error)
	GetRootFolder(user_id string) (*model.FolderModel, error)
	GetFolders(parent_id uuid.UUID) ([]model.FolderModel, error)
	MoveFolder(folder_id, new_parent_id uuid.UUID) error
	RenameFolder(user_id string, folder_id uuid.UUID, new_name string) error
}

type FileRepository interface {
	Create(file *model.FileModel) error
	GetFile(id uuid.UUID) (*model.FileModel, error)
	GetFiles(parent_id uuid.UUID) ([]*model.FileModel, error)
	MoveFile(file_id, new_parent_id uuid.UUID) (string, string, error)
	RenameFile(file_id uuid.UUID, new_name string) error
	RestoreFile(parent_id uuid.UUID, name string) (*model.FileModel, error)
	Update(file *model.FileModel) error
	DeleteFile(id uuid.UUID) error
	ScrubFile(id uuid.UUID) error
}
