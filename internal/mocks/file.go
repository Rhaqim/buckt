package mocks

import (
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/stretchr/testify/mock"
)

type FileService struct {
	mock.Mock
}

var _ domain.FileService = (*FileService)(nil)

// CreateFile implements domain.FileService.
func (m *FileService) CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	args := m.Called(user_id, parent_id, file_name, content_type, file_data)
	return args.String(0), args.Error(1)
}

// GetFile implements domain.FileService.
func (m *FileService) GetFile(file_id string) (*model.FileModel, error) {
	args := m.Called(file_id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *FileService) GetFileStream(file_id string) (*model.FileModel, io.ReadCloser, error) {
	args := m.Called(file_id)

	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}

	return args.Get(0).(*model.FileModel), args.Get(1).(io.ReadCloser), args.Error(2)
}

// GetFiles implements domain.FileService.
func (m *FileService) GetFiles(parent_id string) ([]model.FileModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FileModel), args.Error(1)
}

// MoveFile implements domain.FileService.
func (m *FileService) MoveFile(file_id, new_parent_id string) error {
	args := m.Called(file_id, new_parent_id)
	return args.Error(0)
}

// RenameFile implements domain.FileService.
func (m *FileService) RenameFile(file_id, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

// UpdateFile implements domain.FileService.
func (m *FileService) UpdateFile(user_id, file_id, new_file_name string, new_file_data []byte) error {
	args := m.Called(user_id, file_id, new_file_name, new_file_data)
	return args.Error(0)
}

// DeleteFile implements domain.FileService.
func (m *FileService) DeleteFile(file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}

// ScrubFile implements domain.FileService.
func (m *FileService) ScrubFile(file_id string) (string, error) {
	args := m.Called(file_id)
	return args.String(0), args.Error(1)
}
