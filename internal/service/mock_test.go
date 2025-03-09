package service

import (
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) SetBucktValue(key string, value any) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockCacheManager) GetBucktValue(key string) (any, error) {
	args := m.Called(key)
	return args.Get(0), args.Error(1)
}

func (m *MockCacheManager) DeleteBucktValue(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

type MockFileRepository struct {
	mock.Mock
}

// MoveFile implements domain.FileRepository.
func (m *MockFileRepository) MoveFile(file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	args := m.Called(file_id, new_parent_id)
	return args.Get(0).(string), args.Get(1).(string), args.Error(2)
}

// RenameFile implements domain.FileRepository.
func (m *MockFileRepository) RenameFile(file_id uuid.UUID, new_name string) error {
	args := m.Called(file_id, new_name)
	return args.Error(0)
}

func (m *MockFileRepository) Create(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockFileRepository) GetFile(fileID uuid.UUID) (*model.FileModel, error) {
	args := m.Called(fileID)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

// RestoreFileByPath implements domain.FileRepository.
func (m *MockFileRepository) RestoreFile(path string) (*model.FileModel, error) {
	args := m.Called(path)
	return args.Get(0).(*model.FileModel), args.Error(1)
}

func (m *MockFileRepository) GetFiles(parentID uuid.UUID) ([]*model.FileModel, error) {
	args := m.Called(parentID)
	return args.Get(0).([]*model.FileModel), args.Error(1)
}

func (m *MockFileRepository) Update(file *model.FileModel) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockFileRepository) DeleteFile(userID, fileID uuid.UUID) error {
	args := m.Called(fileID)
	return args.Error(0)
}

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

func (m *MockFolderRepository) RenameFolder(folderID uuid.UUID, newName string) error {
	args := m.Called(folderID, newName)
	return args.Error(0)
}

type MockFolderService struct {
	mock.Mock
}

// GetRootFolder implements domain.FolderService.
func (m *MockFolderService) GetRootFolder(user_id string) (*model.FolderModel, error) {
	args := m.Called(user_id)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

// CreateFolder implements domain.FolderService.
func (m *MockFolderService) CreateFolder(user_id string, parent_id string, folder_name string, description string) (string, error) {
	args := m.Called(user_id, parent_id, folder_name, description)
	return args.Get(0).(string), args.Error(1)
}

// GetFolders implements domain.FolderService.
func (m *MockFolderService) GetFolders(parent_id string) ([]model.FolderModel, error) {
	args := m.Called(parent_id)
	return args.Get(0).([]model.FolderModel), args.Error(1)
}

// MoveFolder implements domain.FolderService.
func (m *MockFolderService) MoveFolder(folder_id string, new_parent_id string) error {
	args := m.Called(folder_id, new_parent_id)
	return args.Error(0)
}

// RenameFolder implements domain.FolderService.
func (m *MockFolderService) RenameFolder(folder_id string, new_name string) error {
	args := m.Called(folder_id, new_name)
	return args.Error(0)
}

func (m *MockFolderService) GetFolder(user_id, folderID string) (*model.FolderModel, error) {
	args := m.Called(user_id, folderID)
	return args.Get(0).(*model.FolderModel), args.Error(1)
}

type MockFileSystemService struct {
	mock.Mock
}

// FSUpdateFile implements domain.FileSystemService.
func (m *MockFileSystemService) FSUpdateFile(oldPath string, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

// FSValidatePath implements domain.FileSystemService.
func (m *MockFileSystemService) FSValidatePath(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockFileSystemService) FSWriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *MockFileSystemService) FSGetFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystemService) FSDeleteFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}
