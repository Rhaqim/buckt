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
	GetTrashFolder(ctx context.Context, user_id string) (*model.FolderModel, error)
	GetFolders(ctx context.Context, parent_id uuid.UUID) ([]model.FolderModel, error)
	GetFoldersPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]model.FolderModel, error)
	MoveFolder(ctx context.Context, folder_id, new_parent_id uuid.UUID) error
	RenameFolder(ctx context.Context, user_id string, folder_id uuid.UUID, new_name string) error
	DeleteFolder(ctx context.Context, folder_id uuid.UUID, beforeCommit func(oldPath, newPath string, fileMoves []model.PathMove) error) (parent_id, oldPath, newPath string, fileMoves []model.PathMove, err error)
	RestoreFolder(ctx context.Context, folder_id, target uuid.UUID, beforeCommit func(oldPath, newPath string, fileMoves []model.PathMove) error) (oldPath, newPath string, fileMoves []model.PathMove, err error)
	ScrubFolder(ctx context.Context, user_id string, folder_id uuid.UUID) (parent_id string, err error)
}

type FileRepository interface {
	Create(ctx context.Context, file *model.FileModel) error
	GetFile(ctx context.Context, id uuid.UUID) (*model.FileModel, error)
	// FindByHash returns an existing file with the given content hash under the
	// same owner and parent folder, or ErrNotFound. Used for content dedup.
	FindByHash(ctx context.Context, user_id string, parent_id uuid.UUID, hash string) (*model.FileModel, error)
	GetFiles(ctx context.Context, parent_id uuid.UUID) ([]*model.FileModel, error)
	GetFilesPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]*model.FileModel, error)
	MoveFile(ctx context.Context, file_id, new_parent_id uuid.UUID) (string, string, error)
	RenameFile(ctx context.Context, file_id uuid.UUID, new_name string) error
	Update(ctx context.Context, file *model.FileModel) error
	DeleteFile(ctx context.Context, id uuid.UUID, beforeCommit func(oldPath, newPath string) error) (oldPath, newPath string, err error)
	RestoreFile(ctx context.Context, id, target uuid.UUID, beforeCommit func(oldPath, newPath string) error) (oldPath, newPath string, err error)
	ScrubFile(ctx context.Context, id uuid.UUID) error
	SetMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error
}
