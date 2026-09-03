package mocks

import (
	"context"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type FileRepository struct {
	mock.Mock
}

var _ domain.FileRepository = (*FileRepository)(nil)

// MoveFile implements domain.FileRepository.
func (m *FileRepository) MoveFile(ctx context.Context, file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	args := m.Called(file_id, new_parent_id)
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// RenameFile implements domain.FileRepository.
func (m *FileRepository) RenameFile(ctx context.Context, file_id uuid.UUID, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

func (m *FileRepository) Create(ctx context.Context, file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *FileRepository) GetFile(ctx context.Context, fileID uuid.UUID) (*model.FileModel, error) {
	args := m.Called(fileID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *FileRepository) GetFiles(ctx context.Context, parentID uuid.UUID) ([]*model.FileModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]*model.FileModel), args.Error(1)
}

func (m *FileRepository) GetFilesPaginated(ctx context.Context, parentID uuid.UUID, page model.Pagination) ([]*model.FileModel, error) {
	args := m.Called(parentID, page)
	return args.Get(0).([]*model.FileModel), args.Error(1)
}

func (m *FileRepository) Update(ctx context.Context, file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *FileRepository) DeleteFile(ctx context.Context, fileID uuid.UUID, beforeCommit func(oldPath, newPath string) error) (string, string, error) {
	args := m.Called(fileID)
	oldPath, newPath := args.String(0), args.String(1)
	repoErr := args.Error(2)

	// Mirror the real repository: the callback only fires after the DB
	// updates succeed. If the mocked call is configured to return an
	// error, treat it as a pre-callback failure and skip the callback.
	if repoErr != nil {
		return "", "", repoErr
	}

	if beforeCommit != nil {
		if cbErr := beforeCommit(oldPath, newPath); cbErr != nil {
			return "", "", cbErr
		}
	}
	return oldPath, newPath, nil
}

func (m *FileRepository) RestoreFile(ctx context.Context, id, target uuid.UUID, beforeCommit func(oldPath, newPath string) error) (string, string, error) {
	args := m.Called(id, target)
	oldPath, newPath := args.String(0), args.String(1)
	repoErr := args.Error(2)

	if repoErr != nil {
		return "", "", repoErr
	}

	if beforeCommit != nil {
		if cbErr := beforeCommit(oldPath, newPath); cbErr != nil {
			return "", "", cbErr
		}
	}
	return oldPath, newPath, nil
}

func (m *FileRepository) FindByHash(ctx context.Context, user_id string, parent_id uuid.UUID, hash string) (*model.FileModel, error) {
	args := m.Called(user_id, parent_id, hash)
	f, _ := args.Get(0).(*model.FileModel)
	return f, args.Error(1)
}

func (m *FileRepository) SetMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	args := m.Called(id, metadata)
	return args.Error(0)
}

func (m *FileRepository) SetExpiry(ctx context.Context, id uuid.UUID, at *time.Time) error {
	args := m.Called(id, at)
	return args.Error(0)
}

func (m *FileRepository) FindExpired(ctx context.Context, now time.Time, limit int) ([]*model.FileModel, error) {
	args := m.Called(now, limit)
	files, _ := args.Get(0).([]*model.FileModel)
	return files, args.Error(1)
}

func (m *FileRepository) FinalizeUpload(ctx context.Context, id uuid.UUID, size int64) (*model.FileModel, error) {
	args := m.Called(id, size)
	f, _ := args.Get(0).(*model.FileModel)
	return f, args.Error(1)
}

func (m *FileRepository) ScrubFile(ctx context.Context, fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}
