package mocks

import (
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockFolderRepository is a mock implementation of the FolderRepository interface
type MockFolderRepository struct {
	mock.Mock
}

// GetRootFolder implements domain.FolderRepository.
func (m *MockFolderRepository) GetRootFolder(user_id string) (*model.FolderModel, error) {
	args := m.Called(user_id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) Create(folder *model.FolderModel) (string, error) {
	args := m.Called(folder)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockFolderRepository) GetFolder(id uuid.UUID) (*model.FolderModel, error) {
	args := m.Called(id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) GetFolders(parentID uuid.UUID) ([]model.FolderModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

func (m *MockFolderRepository) MoveFolder(folderID, newParentID uuid.UUID) error {
	args := m.Called(folderID, newParentID)
	return args.Error(0)
}

func (m *MockFolderRepository) RenameFolder(user_id string, folderID uuid.UUID, newName string) error {
	args := m.Called(user_id, folderID, newName)
	return args.Error(0)
}

// DeleteFolder implements domain.FolderRepository.
func (m *MockFolderRepository) DeleteFolder(folder_id uuid.UUID) (parent_id string, err error) {
	args := m.Called(folder_id)
	return args.String(0), args.Error(1)
}

// ScrubFolder implements domain.FolderRepository.
func (m *MockFolderRepository) ScrubFolder(user_id string, folder_id uuid.UUID) (parent_id string, err error) {
	args := m.Called(user_id, folder_id)
	return args.String(0), args.Error(1)
}
