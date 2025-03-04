package service

import (
	"testing"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileRepository struct {
	mock.Mock
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

func (m *MockFileRepository) DeleteFile(fileID uuid.UUID) error {
	args := m.Called(fileID)
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
func (m *MockFolderService) CreateFolder(user_id string, parent_id string, folder_name string, description string) error {
	args := m.Called(user_id, parent_id, folder_name, description)
	return args.Error(0)
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

func TestCreateFile(t *testing.T) {
	mockLogger := &logger.BucktLogger{}
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, false, mockFileRepo, mockFolderService, mockFileSystemService)

	parentFolder := &model.FolderModel{
		ID:   uuid.New(),
		Path: "/parent/folder",
	}
	mockFolderService.On("GetFolder", "parent_id").Return(parentFolder, nil)
	mockFileSystemService.On("FSWriteFile", "/parent/folder/file.txt", []byte("file data")).Return(nil)
	mockFileRepo.On("Create", mock.Anything).Return(nil)

	_, err := fileService.CreateFile("user_123", "parent_id", "file.txt", "text/plain", []byte("file data"))
	assert.NoError(t, err)
}

func TestGetFile(t *testing.T) {
	mockLogger := &logger.BucktLogger{}
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, false, mockFileRepo, mockFolderService, mockFileSystemService)

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	mockFileRepo.On("GetFile", fileID).Return(fileModel, nil)
	mockFileSystemService.On("FSGetFile", "/parent/folder/file.txt").Return([]byte("file data"), nil)

	file, err := fileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFiles(t *testing.T) {
	mockLogger := &logger.BucktLogger{}
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, false, mockFileRepo, mockFolderService, mockFileSystemService)

	parentID := uuid.New()
	fileModels := []model.FileModel{
		{ID: uuid.New(), Path: "/parent/folder/file1.txt"},
		{ID: uuid.New(), Path: "/parent/folder/file2.txt"},
	}
	mockFileRepo.On("GetFiles", parentID).Return(fileModels, nil)
	mockFileSystemService.On("FSGetFile", "/parent/folder/file1.txt").Return([]byte("file1 data"), nil)
	mockFileSystemService.On("FSGetFile", "/parent/folder/file2.txt").Return([]byte("file2 data"), nil)

	files, err := fileService.GetFiles(parentID.String())
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, []byte("file1 data"), files[0].Data)
	assert.Equal(t, []byte("file2 data"), files[1].Data)
}

func TestUpdateFile(t *testing.T) {
	mockLogger := &logger.BucktLogger{}
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, false, mockFileRepo, mockFolderService, mockFileSystemService)

	fileID := uuid.New()
	parentID := uuid.New()
	fileModel := &model.FileModel{
		ID:       fileID,
		ParentID: parentID,
		Path:     "/parent/folder/file.txt",
	}
	parentFolder := &model.FolderModel{
		ID:   fileModel.ParentID,
		Path: "/parent/folder",
	}
	mockFileRepo.On("GetFile", fileID).Return(fileModel, nil)
	mockFolderService.On("GetFolder", fileModel.ParentID.String()).Return(parentFolder, nil)
	mockFileSystemService.On("FSWriteFile", "/parent/folder/new_file.txt", []byte("new file data")).Return(nil)
	mockFileRepo.On("Update", mock.Anything).Return(nil)

	err := fileService.UpdateFile(fileID.String(), "new_file.txt", []byte("new file data"))
	assert.NoError(t, err)
}

func TestDeleteFile(t *testing.T) {
	mockLogger := &logger.BucktLogger{}
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, false, mockFileRepo, mockFolderService, mockFileSystemService)

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	mockFileRepo.On("GetFile", fileID).Return(fileModel, nil)
	mockFileSystemService.On("FSDeleteFile", "/parent/folder/file.txt").Return(nil)
	mockFileRepo.On("DeleteFile", fileID).Return(nil)

	_, err := fileService.DeleteFile(fileID.String())
	assert.NoError(t, err)
}
