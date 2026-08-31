package domain

import (
	"context"
	"io"
	"time"

	"github.com/Rhaqim/buckt/internal/model"
)

type FolderService interface {
	CreateFolder(ctx context.Context, user_id, parent_id, folder_name, description string) (string, error)
	GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error)
	GetTrashFolder(ctx context.Context, user_id string) (*model.FolderModel, error)
	GetFolder(ctx context.Context, user_id, folder_id string) (*model.FolderModel, error)
	GetFolders(ctx context.Context, parent_id string) ([]model.FolderModel, error)
	MoveFolder(ctx context.Context, folder_id, new_parent_id string) error
	RenameFolder(ctx context.Context, user_id, folder_id, new_name string) error
	DeleteFolder(ctx context.Context, folder_id string) (string, error)
	RestoreFolder(ctx context.Context, user_id, folder_id string) error
	ScrubFolder(ctx context.Context, user_id, folder_id string) (string, error)
}

type FileService interface {
	CreateFile(ctx context.Context, user_id, parent_id, file_name, content_type string, file_data []byte) (string, error)
	GetFile(ctx context.Context, file_id string) (*model.FileModel, error)
	GetFileStream(ctx context.Context, file_id string) (*model.FileModel, io.ReadCloser, error)
	GetFiles(ctx context.Context, parent_id string) ([]model.FileModel, error)
	GetFilesMetadata(ctx context.Context, parent_id string) ([]model.FileModel, error)
	MoveFile(ctx context.Context, file_id, new_parent_id string) error
	RenameFile(ctx context.Context, file_id, new_name string) error
	UpdateFile(ctx context.Context, user_id, file_id, new_file_name string, new_file_data []byte) error
	DeleteFile(ctx context.Context, file_id string) (string, error)
	RestoreFile(ctx context.Context, user_id, file_id string) error
	ScrubFile(ctx context.Context, file_id string) (string, error)
	SetMetadata(ctx context.Context, file_id string, metadata map[string]string) error
	GetMetadata(ctx context.Context, file_id string) (map[string]string, error)
	// SetExpiry sets (or, with a nil time, clears) a file's automatic-deletion time.
	SetExpiry(ctx context.Context, file_id string, at *time.Time) error
	// FindExpired returns up to limit files whose expiry is at or before now.
	FindExpired(ctx context.Context, now time.Time, limit int) ([]*model.FileModel, error)
}
