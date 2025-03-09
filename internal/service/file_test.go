package service

import (
	"testing"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileServices struct {
	*FileService
	*MockCacheManager
	*MockFileRepository
	*MockFolderService
	*MockFileSystemService
}

func setupFileTest() MockFileServices {
	mockLogger := logger.NewLogger("test", true)
	mockCache := new(MockCacheManager)
	mockFileRepo := new(MockFileRepository)
	mockFolderService := new(MockFolderService)
	mockFileSystemService := new(MockFileSystemService)

	fileService := NewFileService(mockLogger, mockCache, false, mockFileRepo, mockFolderService, mockFileSystemService).(*FileService)

	return MockFileServices{
		FileService:           fileService,
		MockCacheManager:      mockCache,
		MockFileRepository:    mockFileRepo,
		MockFolderService:     mockFolderService,
		MockFileSystemService: mockFileSystemService,
	}
}

func TestCreateFile(t *testing.T) {
	mockSetUp := setupFileTest()

	parentFolder := &model.FolderModel{
		ID:   uuid.New(),
		Path: "/parent/folder",
	}

	user_id := "user1"

	// Mock GetFolder to match the actual method call
	mockSetUp.MockFolderService.On("GetFolder", user_id, "parent_id").Return(parentFolder, nil)

	// Mock FSWriteFile
	mockSetUp.MockFileSystemService.On("FSWriteFile", "/parent/folder/file.txt", []byte("file data")).Return(nil)

	// Mock Create
	mockSetUp.MockFileRepository.On("Create", mock.Anything).Return(nil)

	_, err := mockSetUp.FileService.CreateFile(user_id, "parent_id", "file.txt", "text/plain", []byte("file data"))
	assert.NoError(t, err)
}

func TestGetFile(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}

	var jsonStr string

	mockSetUp.MockCacheManager.On("GetBucktValue", fileID.String()).Return(jsonStr, nil)

	mockSetUp.MockCacheManager.On("SetBucktValue", fileID.String(), mock.Anything).Return(nil)

	mockSetUp.MockFileRepository.On("GetFile", fileID).Return(fileModel, nil)

	mockSetUp.MockFileSystemService.On("FSGetFile", "/parent/folder/file.txt").Return([]byte("file data"), nil)

	file, err := mockSetUp.FileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFiles(t *testing.T) {
	mockSetUp := setupFileTest()

	parentID := uuid.New()
	fileModels := []*model.FileModel{
		{ID: uuid.New(), Path: "/parent/folder/file1.txt"},
		{ID: uuid.New(), Path: "/parent/folder/file2.txt"},
	}

	var jsonStr string

	mockSetUp.MockCacheManager.On("GetBucktValue", "files:"+parentID.String()).Return(jsonStr, nil)

	mockSetUp.MockCacheManager.On("SetBucktValue", "files:"+parentID.String(), mock.Anything).Return(nil)

	mockSetUp.MockFileRepository.On("GetFiles", parentID).Return(fileModels, nil)

	mockSetUp.MockFileSystemService.On("FSGetFile", "/parent/folder/file1.txt").Return([]byte("file1 data"), nil)

	mockSetUp.MockFileSystemService.On("FSGetFile", "/parent/folder/file2.txt").Return([]byte("file2 data"), nil)

	files, err := mockSetUp.FileService.GetFiles(parentID.String())
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, []byte("file1 data"), files[0].Data)
	assert.Equal(t, []byte("file2 data"), files[1].Data)
}

func TestUpdateFile(t *testing.T) {
	mockSetUp := setupFileTest()

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

	user_id := "user1"

	mockSetUp.MockFileRepository.On("GetFile", fileID).Return(fileModel, nil)

	mockSetUp.MockFolderService.On("GetFolder", user_id, parentID.String()).Return(parentFolder, nil)

	mockSetUp.MockFileSystemService.On("FSWriteFile", "/parent/folder/new_file.txt", []byte("new file data")).Return(nil)

	mockSetUp.MockFileRepository.On("Update", mock.Anything).Return(nil)

	err := mockSetUp.FileService.UpdateFile(user_id, fileID.String(), "new_file.txt", []byte("new file data"))
	assert.NoError(t, err)
}

func TestDeleteFile(t *testing.T) {
	mockSetUp := setupFileTest()

	userID := uuid.New()
	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}

	mockSetUp.MockFileRepository.On("GetFile", fileID).Return(fileModel, nil)

	mockSetUp.MockFileSystemService.On("FSDeleteFile", "/parent/folder/file.txt").Return(nil)

	mockSetUp.MockFileRepository.On("DeleteFile", userID, fileID).Return(nil)

	_, err := mockSetUp.FileService.DeleteFile(userID.String(), fileID.String())
	assert.NoError(t, err)
}
