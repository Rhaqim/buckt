package service

import (
	"encoding/json"
	"testing"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFolderServices struct {
	folderService    domain.FolderService
	cacheManager     *mocks.CacheManager
	folderRepository *mocks.FolderRepository
	backend          *mocks.LocalFileSystemService
}

func setupFolderTest() MockFolderServices {
	return setupFolderTestWithMode(true)
}

func setupFolderTestWithMode(flatNamespaces bool) MockFolderServices {
	mockLogger := logger.NewLogger("", true, false)
	mockCache := new(mocks.CacheManager)
	mockFolderRepo := new(mocks.FolderRepository)
	mockFileSystemService := new(mocks.LocalFileSystemService)

	folderService := NewFolderService(mockLogger, mockCache, mockFolderRepo, mockFileSystemService, flatNamespaces, 0, 0)

	return MockFolderServices{
		folderService:    folderService,
		cacheManager:     mockCache,
		folderRepository: mockFolderRepo,
		backend:          mockFileSystemService,
	}
}

func TestCreateFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	// Define the expected folder return for GetFolder
	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Mock GetFolder to return a valid folder
	mockSetUp.folderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock Create method
	mockSetUp.folderRepository.On("Create", mock.Anything).Return(folderID.String(), nil)

	new_folder_id, err := mockSetUp.folderService.CreateFolder(ctx, "user1", folderID.String(), "folder", "description")

	assert.Len(t, new_folder_id, 36)

	assert.NoError(t, err)

	mockSetUp.folderRepository.AssertExpectations(t)
}

func TestGetFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolder := &model.FolderModel{ID: folderID, Name: "folder"}

	// Marshal mockFolder to JSON string
	jsonBytes, _ := json.Marshal(mockFolder)
	jsonStr := string(jsonBytes)

	// Mock cache get (simulate cache miss)
	mockSetUp.cacheManager.On("GetBucktValue", "folder:"+folderID.String()).Return("", nil)

	// Mock repo get
	mockSetUp.folderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	// Mock cache set with correct JSON string
	mockSetUp.cacheManager.On("SetBucktValue", "folder:"+folderID.String(), jsonStr).Return(nil)

	folder, err := mockSetUp.folderService.GetFolder(ctx, "user1", folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolder, folder)

	mockSetUp.folderRepository.AssertExpectations(t)
	mockSetUp.cacheManager.AssertExpectations(t)
}

func TestGetFolders(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	mockFolders := []model.FolderModel{
		{ID: uuid.New(), Name: "folder1"},
		{ID: uuid.New(), Name: "folder2"},
	}

	mockSetUp.folderRepository.On("GetFolders", parentID).Return(mockFolders, nil)

	folders, err := mockSetUp.folderService.GetFolders(ctx, parentID.String())
	assert.NoError(t, err)
	assert.Equal(t, mockFolders, folders)
	mockSetUp.folderRepository.AssertExpectations(t)
}

func TestMoveFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newParentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mockSetUp.folderRepository.On("MoveFolder", folderID, newParentID).Return(nil)

	err := mockSetUp.folderService.MoveFolder(ctx, folderID.String(), newParentID.String())
	assert.NoError(t, err)
	mockSetUp.folderRepository.AssertExpectations(t)
}

func TestRenameFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	newName := "new_folder_name"

	user_id := "user1"

	mockSetUp.folderRepository.On("RenameFolder", user_id, folderID, newName).Return(nil)

	err := mockSetUp.folderService.RenameFolder(ctx, user_id, folderID.String(), newName)
	assert.NoError(t, err)
	mockSetUp.folderRepository.AssertExpectations(t)
}

func TestDeleteFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	mockSetUp.folderRepository.On("DeleteFolder", folderID).Return(parentID.String(), "", "", nil, nil)

	returnedParentID, err := mockSetUp.folderService.DeleteFolder(ctx, folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)
	mockSetUp.folderRepository.AssertExpectations(t)
}

// TestDeleteFolder_NestedMoveToTrash verifies that in nested-namespace mode,
// the service physically moves every descendant file on the backend to
// follow the DB path rewrite.
func TestDeleteFolder_NestedMoveToTrash(t *testing.T) {
	mockSetUp := setupFolderTestWithMode(false) // nested mode
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	oldPath := "/user/photos"
	newPath := "/user/__trash__/photos"

	fileMoves := []model.PathMove{
		{Old: "/user/photos/sunset.jpg", New: "/user/__trash__/photos/sunset.jpg"},
		{Old: "/user/photos/beach.jpg", New: "/user/__trash__/photos/beach.jpg"},
	}

	mockSetUp.folderRepository.On("DeleteFolder", folderID).
		Return(parentID.String(), oldPath, newPath, fileMoves, nil)

	// Each descendant file should be physically moved on the backend
	for _, mv := range fileMoves {
		mockSetUp.backend.On("Move", mv.Old, mv.New).Return(nil)
	}

	returnedParentID, err := mockSetUp.folderService.DeleteFolder(ctx, folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)

	for _, mv := range fileMoves {
		mockSetUp.backend.AssertCalled(t, "Move", mv.Old, mv.New)
	}
	mockSetUp.backend.AssertNotCalled(t, "DeleteFolder", mock.Anything)
}

// TestDeleteFolder_NestedPermanentDelete verifies that when the repo
// returns newPath == "" (already in trash -> hard delete), the service
// calls backend.DeleteFolder on the old prefix.
func TestDeleteFolder_NestedPermanentDelete(t *testing.T) {
	mockSetUp := setupFolderTestWithMode(false) // nested mode
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	oldPath := "/user/__trash__/photos"

	// newPath == "" signals permanent delete
	mockSetUp.folderRepository.On("DeleteFolder", folderID).
		Return(parentID.String(), oldPath, "", nil, nil)

	mockSetUp.backend.On("DeleteFolder", oldPath).Return(nil)

	returnedParentID, err := mockSetUp.folderService.DeleteFolder(ctx, folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)
	mockSetUp.backend.AssertCalled(t, "DeleteFolder", oldPath)
	mockSetUp.backend.AssertNotCalled(t, "Move", mock.Anything, mock.Anything)
}

func TestScrubFolder(t *testing.T) {
	mockSetUp := setupFolderTest()
	ctx := t.Context()

	folderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	mockFolder := &model.FolderModel{ID: folderID, Path: "/path/to/folder"}

	mockSetUp.folderRepository.On("GetFolder", folderID).Return(mockFolder, nil)

	mockSetUp.backend.On("DeleteFolder", mockFolder.Path).Return(nil)

	mockSetUp.folderRepository.On("ScrubFolder", "user1", folderID).Return(parentID.String(), nil)

	returnedParentID, err := mockSetUp.folderService.ScrubFolder(ctx, "user1", folderID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), returnedParentID)
	mockSetUp.folderRepository.AssertExpectations(t)
	mockSetUp.backend.AssertExpectations(t)
}
