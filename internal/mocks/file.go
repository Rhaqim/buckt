package mocks

import (
	"context"
	"io"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/stretchr/testify/mock"
)

type FileService struct {
	mock.Mock
}

var _ domain.FileService = (*FileService)(nil)

// CreateFile implements domain.FileService.
func (m *FileService) CreateFile(ctx context.Context, user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	args := m.Called(user_id, parent_id, file_name, content_type, file_data)
	return args.String(0), args.Error(1)
}

func (m *FileService) CreateFileWithExpiry(ctx context.Context, user_id, parent_id, file_name, content_type string, file_data []byte, expires_at *time.Time) (string, error) {
	args := m.Called(user_id, parent_id, file_name, content_type, file_data, expires_at)
	return args.String(0), args.Error(1)
}

func (m *FileService) GetFilesMetadata(ctx context.Context, parent_id string) ([]model.FileModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

// GetFile implements domain.FileService.
func (m *FileService) GetFile(ctx context.Context, file_id string) (*model.FileModel, error) {
	args := m.Called(file_id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *FileService) GetFileStream(ctx context.Context, file_id string) (*model.FileModel, io.ReadCloser, error) {
	args := m.Called(file_id)

	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}

	return args.Get(0).(*model.FileModel), args.Get(1).(io.ReadCloser), args.Error(2)
}

// GetFiles implements domain.FileService.
func (m *FileService) GetFiles(ctx context.Context, parent_id string) ([]model.FileModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

// MoveFile implements domain.FileService.
func (m *FileService) MoveFile(ctx context.Context, file_id, new_parent_id string) error {
	args := m.Called(file_id, new_parent_id)
	return args.Error(0)
}

// RenameFile implements domain.FileService.
func (m *FileService) RenameFile(ctx context.Context, file_id, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

// UpdateFile implements domain.FileService.
func (m *FileService) UpdateFile(ctx context.Context, user_id, file_id, new_file_name string, new_file_data []byte) error {
	args := m.Called(user_id, file_id, new_file_name, new_file_data)
	return args.Error(0)
}

// DeleteFile implements domain.FileService.
func (m *FileService) DeleteFile(ctx context.Context, file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}

// RestoreFile implements domain.FileService.
func (m *FileService) RestoreFile(ctx context.Context, user_id, file_id string) error {
	args := m.Called(user_id, file_id)
	return args.Error(0)
}

// SetMetadata implements domain.FileService.
func (m *FileService) SetMetadata(ctx context.Context, file_id string, metadata map[string]string) error {
	args := m.Called(file_id, metadata)
	return args.Error(0)
}

// SetExpiry implements domain.FileService.
func (m *FileService) SetExpiry(ctx context.Context, file_id string, at *time.Time) error {
	args := m.Called(file_id, at)
	return args.Error(0)
}

// FindExpired implements domain.FileService.
func (m *FileService) FindExpired(ctx context.Context, now time.Time, limit int) ([]*model.FileModel, error) {
	args := m.Called(now, limit)
	files, _ := args.Get(0).([]*model.FileModel)
	return files, args.Error(1)
}

// CreatePendingUpload implements domain.FileService.
func (m *FileService) CreatePendingUpload(ctx context.Context, user_id, parent_id, file_name, content_type string) (string, string, error) {
	args := m.Called(user_id, parent_id, file_name, content_type)
	return args.String(0), args.String(1), args.Error(2)
}

// FinalizeUpload implements domain.FileService.
func (m *FileService) FinalizeUpload(ctx context.Context, file_id string, size int64) (string, error) {
	args := m.Called(file_id, size)
	return args.String(0), args.Error(1)
}

// GetMetadata implements domain.FileService.
func (m *FileService) GetMetadata(ctx context.Context, file_id string) (map[string]string, error) {
	args := m.Called(file_id)
	meta, _ := args.Get(0).(map[string]string)
	return meta, args.Error(1)
}

// ScrubFile implements domain.FileService.
func (m *FileService) ScrubFile(ctx context.Context, file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}
