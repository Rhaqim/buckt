package service

import (
	"encoding/json"
	"testing"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func (m *MockFolderRepository) Create(folder *model.FolderModel) error {
	args := m.Called(folder)
	return args.Error(0)
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

func TestCreateFolder(t *testing.T) {
	mockRepo := new(MockFolderRepository)
	mockCache := new(MockCacheManager)
	log := logger.NewLogger("test", true)
	service := NewFolderService(log, mockCache, mockRepo)

	// Define the expected folder return for GetFolder
	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Mock GetFolder to return a valid folder
	mockRepo.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock Create method
	mockRepo.On("Create", mock.Anything).Return(nil)

	err := service.CreateFolder("user1", folderID.String(), "folder", "description")
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestGetFolder(t *testing.T) {
	mockRepo := new(MockFolderRepository)
	mockCache := new(MockCacheManager)
	log := logger.NewLogger("test", true)
	service := NewFolderService(log, mockCache, mockRepo)

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Marshal mockFolder to JSON string
	jsonBytes, _ := json.Marshal(mockFolder)
	jsonStr := string(jsonBytes)

	// Mock cache get (simulate cache miss)
	mockCache.On("Get", "folder:"+folderID.String()).Return("", nil)

	// Mock repo get
	mockRepo.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock cache set with correct JSON string
	mockCache.On("Set", "folder:"+folderID.String(), jsonStr).Return(nil)

	folder, err := service.GetFolder("user1", folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolder, folder)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetFolders(t *testing.T) {
	mockRepo := new(MockFolderRepository)
	mockCache := new(MockCacheManager)
	log := logger.NewLogger("test", true)
	service := NewFolderService(log, mockCache, mockRepo)

	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolders := []model.FolderModel{
		{ID: uuid.New(), Name: "folder1"},
		{ID: uuid.New(), Name: "folder2"},
	}

	mockRepo.On("GetFolders", parentID).Return(mockFolders, nil)

	folders, err := service.GetFolders(parentID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolders, folders)
	mockRepo.AssertExpectations(t)
}

func TestMoveFolder(t *testing.T) {
	mockRepo := new(MockFolderRepository)
	mockCache := new(MockCacheManager)
	log := logger.NewLogger("test", true)
	service := NewFolderService(log, mockCache, mockRepo)

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newParentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mockRepo.On("MoveFolder", folderID, newParentID).Return(nil)

	err := service.MoveFolder(folderID.String(), newParentID.String())
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRenameFolder(t *testing.T) {
	mockRepo := new(MockFolderRepository)
	mockCache := new(MockCacheManager)
	log := logger.NewLogger("test", true)
	service := NewFolderService(log, mockCache, mockRepo)

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newName := "new_folder_name"

	mockRepo.On("RenameFolder", folderID, newName).Return(nil)

	err := service.RenameFolder(folderID.String(), newName)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
