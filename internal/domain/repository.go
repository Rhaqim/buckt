package domain

import (
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type FolderRepository interface {
	Create(folder *model.FolderModel) error
	GetFolder(folder_id uuid.UUID) (*model.FolderModel, error)
	GetRootFolder(user_id string) (*model.FolderModel, error)
	GetFolders(bucket_id uuid.UUID) ([]model.FolderModel, error)
	MoveFolder(folder_id, new_parent_id uuid.UUID) error
	RenameFolder(folder_id uuid.UUID, new_name string) error
}

type FileRepository interface {
	Create(file *model.FileModel) error
	Update(file *model.FileModel) error
	GetFile(id uuid.UUID) (*model.FileModel, error)
	RestoreFile(hash string) (*model.FileModel, error)
	GetFiles(parent_id uuid.UUID) ([]*model.FileModel, error)
	DeleteFile(id uuid.UUID) error
}
