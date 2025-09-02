package mocks

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/stretchr/testify/mock"
)

type FolderService struct {
	mock.Mock
}

func NewFolderService() domain.FolderService {
	return &FolderService{}
}

// GetRootFolder implements domain.FolderService.
func (m *FolderService) GetRootFolder(user_id string) (*model.FolderModel, error) {
	args := m.Called(user_id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

// CreateFolder implements domain.FolderService.
func (m *FolderService) CreateFolder(user_id string, parent_id string, folder_name string, description string) (string, error) {
	args := m.Called(user_id, parent_id, folder_name, description)
	return args.Get(0).(string), args.Error(1)
}

// GetFolders implements domain.FolderService.
func (m *FolderService) GetFolders(parent_id string) ([]model.FolderModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

// MoveFolder implements domain.FolderService.
func (m *FolderService) MoveFolder(folder_id string, new_parent_id string) error {
	args := m.Called(folder_id, new_parent_id)
	return args.Error(0)
}

// RenameFolder implements domain.FolderService.
func (m *FolderService) RenameFolder(user_id, folder_id string, new_name string) error {
	args := m.Called(user_id, folder_id, new_name)
	return args.Error(0)
}

func (m *FolderService) GetFolder(user_id, folderID string) (*model.FolderModel, error) {
	args := m.Called(user_id, folderID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.FolderModel), args.Error(1)
}

// DeleteFolder implements domain.FolderService.
func (m *FolderService) DeleteFolder(folder_id string) (string, error) {
	args := m.Called(folder_id)
	return args.String(0), args.Error(1)
}

// ScrubFolder implements domain.FolderService.
func (m *FolderService) ScrubFolder(user_id string, folder_id string) (string, error) {
	args := m.Called(user_id, folder_id)
	return args.String(0), args.Error(1)
}
