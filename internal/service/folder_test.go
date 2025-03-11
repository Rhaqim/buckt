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

type MockFolderServices struct {
	*FolderService
	*MockCacheManager
	*MockFolderRepository
	*MockFileSystemService
}

func setupFolderTest() MockFolderServices {
	mockLogger := logger.NewLogger("", true)
	mockCache := new(MockCacheManager)
	mockFolderRepo := new(MockFolderRepository)
	mockFileSystemService := new(MockFileSystemService)

	folderService := NewFolderService(mockLogger, mockCache, mockFolderRepo, mockFileSystemService).(*FolderService)

	return MockFolderServices{
		FolderService:         folderService,
		MockCacheManager:      mockCache,
		MockFolderRepository:  mockFolderRepo,
		MockFileSystemService: mockFileSystemService,
	}
}

func TestCreateFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	// Define the expected folder return for GetFolder
	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Mock GetFolder to return a valid folder
	mockSetUp.MockFolderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock Create method
	mockSetUp.MockFolderRepository.On("Create", mock.Anything).Return(folderID.String(), nil)

	new_folder_id, err := mockSetUp.FolderService.CreateFolder("user1", folderID.String(), "folder", "description")

	assert.Len(t, new_folder_id, 36)

	assert.NoError(t, err)

	mockSetUp.MockFolderRepository.AssertExpectations(t)
}

func TestGetFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Marshal mockFolder to JSON string
	jsonBytes, _ := json.Marshal(mockFolder)
	jsonStr := string(jsonBytes)

	// Mock cache get (simulate cache miss)
	mockSetUp.MockCacheManager.On("GetBucktValue", "folder:"+folderID.String()).Return("", nil)

	// Mock repo get
	mockSetUp.MockFolderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock cache set with correct JSON string
	mockSetUp.MockCacheManager.On("SetBucktValue", "folder:"+folderID.String(), jsonStr).Return(nil)

	folder, err := mockSetUp.FolderService.GetFolder("user1", folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolder, folder)

	mockSetUp.MockFolderRepository.AssertExpectations(t)
	mockSetUp.MockCacheManager.AssertExpectations(t)
}

func TestGetFolders(t *testing.T) {
	mockSetUp := setupFolderTest()

	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolders := []model.FolderModel{
		{ID: uuid.New(), Name: "folder1"},
		{ID: uuid.New(), Name: "folder2"},
	}

	mockSetUp.MockFolderRepository.On("GetFolders", parentID).Return(mockFolders, nil)

	folders, err := mockSetUp.FolderService.GetFolders(parentID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolders, folders)
	mockSetUp.MockFolderRepository.AssertExpectations(t)
}

func TestMoveFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newParentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mockSetUp.MockFolderRepository.On("MoveFolder", folderID, newParentID).Return(nil)

	err := mockSetUp.FolderService.MoveFolder(folderID.String(), newParentID.String())
	assert.NoError(t, err)
	mockSetUp.MockFolderRepository.AssertExpectations(t)
}

func TestRenameFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newName := "new_folder_name"

	user_id := "user1"

	mockSetUp.MockFolderRepository.On("RenameFolder", user_id, folderID, newName).Return(nil)

	err := mockSetUp.FolderService.RenameFolder(user_id, folderID.String(), newName)
	assert.NoError(t, err)
	mockSetUp.MockFolderRepository.AssertExpectations(t)
}

func TestDeleteFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mockSetUp.MockFolderRepository.On("DeleteFolder", folderID).Return(parentID.String(), nil)

	returnedParentID, err := mockSetUp.FolderService.DeleteFolder(folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)
	mockSetUp.MockFolderRepository.AssertExpectations(t)
}

func TestScrubFolder(t *testing.T) {
	mockSetUp := setupFolderTest()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	mockFolder := &model.FolderModel{ID: folderID, Path: "/path/to/folder"}

	mockSetUp.MockFolderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	mockSetUp.MockFileSystemService.On("FSDeleteFolder", mockFolder.Path).Return(nil)

	mockSetUp.MockFolderRepository.On("ScrubFolder", "user1", folderID).Return(parentID.String(), nil)

	returnedParentID, err := mockSetUp.FolderService.ScrubFolder("user1", folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)
	mockSetUp.MockFolderRepository.AssertExpectations(t)
	mockSetUp.MockFileSystemService.AssertExpectations(t)
}
