package mocks

import (
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
func (m *FileRepository) MoveFile(file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	args := m.Called(file_id, new_parent_id)
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// RenameFile implements domain.FileRepository.
func (m *FileRepository) RenameFile(file_id uuid.UUID, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

func (m *FileRepository) Create(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *FileRepository) GetFile(fileID uuid.UUID) (*model.FileModel, error) {
	args := m.Called(fileID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FileModel), args.Error(1)
}

// RestoreFileByPath implements domain.FileRepository.
func (m *FileRepository) RestoreFile(parent_id uuid.UUID, name string) (*model.FileModel, error) {
	args := m.Called(parent_id, name)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *FileRepository) GetFiles(parentID uuid.UUID) ([]*model.FileModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]*model.FileModel), args.Error(1)
}

func (m *FileRepository) Update(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *FileRepository) DeleteFile(fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}

func (m *FileRepository) ScrubFile(fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}
