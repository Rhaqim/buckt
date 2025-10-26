package domain

import (
	"context"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type FolderRepository interface {
	Create(ctx context.Context, folder *model.FolderModel) (string, error)
	GetFolder(ctx context.Context, folder_id uuid.UUID) (*model.FolderModel, error)
	GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error)
	GetFolders(ctx context.Context, parent_id uuid.UUID) ([]model.FolderModel, error)
	MoveFolder(ctx context.Context, folder_id, new_parent_id uuid.UUID) error
	RenameFolder(ctx context.Context, user_id string, folder_id uuid.UUID, new_name string) error
	DeleteFolder(ctx context.Context, folder_id uuid.UUID) (parent_id string, err error)
	ScrubFolder(ctx context.Context, user_id string, folder_id uuid.UUID) (parent_id string, err error)
}

type FileRepository interface {
	Create(ctx context.Context, file *model.FileModel) error
	GetFile(ctx context.Context, id uuid.UUID) (*model.FileModel, error)
	GetFiles(ctx context.Context, parent_id uuid.UUID) ([]*model.FileModel, error)
	MoveFile(ctx context.Context, file_id, new_parent_id uuid.UUID) (string, string, error)
	RenameFile(ctx context.Context, file_id uuid.UUID, new_name string) error
	RestoreFile(ctx context.Context, parent_id uuid.UUID, name string) (*model.FileModel, error)
	Update(ctx context.Context, file *model.FileModel) error
	DeleteFile(ctx context.Context, id uuid.UUID) error
	ScrubFile(ctx context.Context, id uuid.UUID) error
}
