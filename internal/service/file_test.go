package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFileServices struct {
	fileService    domain.FileService
	cacheManager   *mocks.CacheManager
	fileRepository *mocks.FileRepository
	folderService  *mocks.FolderService
	backend        *mocks.LocalFileSystemService
}

func setupFileTest() MockFileServices {
	mockLogger := logger.NewLogger("", true, false)
	mockCache := new(mocks.CacheManager)
	mockFileRepo := new(mocks.FileRepository)
	mockFolderService := new(mocks.FolderService)
	mockLocalFileSystemService := new(mocks.LocalFileSystemService)

	fileService := NewFileService(mockLogger, mockCache, mockFileRepo, mockFolderService, mockLocalFileSystemService, false)

	return MockFileServices{
		fileService:    fileService,
		cacheManager:   mockCache,
		fileRepository: mockFileRepo,
		folderService:  mockFolderService,
		backend:        mockLocalFileSystemService,
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
	mockSetUp.folderService.On("GetFolder", user_id, "parent_id").Return(parentFolder, nil)

	// Mock Put
	mockSetUp.backend.On("Put", "/parent/folder/file.txt", []byte("file data")).Return(nil)

	// Mock Create
	mockSetUp.fileRepository.On("Create", mock.Anything).Return(nil)

	_, err := mockSetUp.fileService.CreateFile(user_id, "parent_id", "file.txt", "text/plain", []byte("file data"))
	assert.NoError(t, err)
}

func TestGetFiles(t *testing.T) {
	mockSetUp := setupFileTest()

	parentID := uuid.New()
	fileModels := []*model.FileModel{
		{ID: uuid.New(), Path: "/parent/folder/file1.txt"},
		{ID: uuid.New(), Path: "/parent/folder/file2.txt"},
	}

	var jsonStr string

	mockSetUp.cacheManager.On("GetBucktValue", "files:"+parentID.String()).Return(jsonStr, nil)

	mockSetUp.cacheManager.On("SetBucktValue", "files:"+parentID.String(), mock.Anything).Return(nil)

	mockSetUp.fileRepository.On("GetFiles", parentID).Return(fileModels, nil)

	mockSetUp.backend.On("Get", "/parent/folder/file1.txt").Return([]byte("file1 data"), nil)

	mockSetUp.backend.On("Get", "/parent/folder/file2.txt").Return([]byte("file2 data"), nil)

	files, err := mockSetUp.fileService.GetFiles(parentID.String())
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

	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	mockSetUp.folderService.On("GetFolder", user_id, parentID.String()).Return(parentFolder, nil)

	mockSetUp.backend.On("Put", "/parent/folder/new_file.txt", []byte("new file data")).Return(nil)

	mockSetUp.fileRepository.On("Update", mock.Anything).Return(nil)

	err := mockSetUp.fileService.UpdateFile(user_id, fileID.String(), "new_file.txt", []byte("new file data"))
	assert.NoError(t, err)
}

func TestDeleteFile(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return("", nil)

	// Mock cache deletion
	mockSetUp.cacheManager.On("DeleteBucktValue", fileID.String()).Return(nil)

	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	mockSetUp.backend.On("Delete", "/parent/folder/file.txt").Return(nil)

	mockSetUp.fileRepository.On("DeleteFile", fileID).Return(nil)

	_, err := mockSetUp.fileService.DeleteFile(fileID.String())
	assert.NoError(t, err)
}

func TestScrubFile(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	parentID := uuid.New()
	fileModel := &model.FileModel{
		ID:       fileID,
		ParentID: parentID,
		Path:     "/parent/folder/file.txt",
	}

	var jsonStr string

	// Mock cache retrieval
	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(jsonStr, nil)

	// Mock cache deletion
	mockSetUp.cacheManager.On("DeleteBucktValue", fileID.String()).Return(nil)

	// Mock repository retrieval
	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	// Mock file system deletion
	mockSetUp.backend.On("Delete", "/parent/folder/file.txt").Return(nil)

	// Mock repository scrub
	mockSetUp.fileRepository.On("ScrubFile", fileID).Return(nil)

	parentIDStr, err := mockSetUp.fileService.ScrubFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), parentIDStr)
}

func TestGetFile_CacheHit(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	jsonData, _ := json.Marshal(fileModel)

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(string(jsonData), nil)

	mockSetUp.cacheManager.On("GetBucktValue", fileModel.Path).Return(nil, nil)

	mockSetUp.backend.On("Get", fileModel.Path).Return([]byte("file data"), nil)

	mockSetUp.cacheManager.On("SetBucktValue", fileModel.Path, []byte("file data")).Return(nil)

	file, err := mockSetUp.fileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, fileModel.Path, file.Path)
}

func TestGetFile_CacheMiss_RepoHit(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(nil, nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)
	mockSetUp.cacheManager.On("SetBucktValue", fileID.String(), mock.Anything).Return(nil)
	mockSetUp.backend.On("Get", fileModel.Path).Return([]byte("file data"), nil)
	mockSetUp.cacheManager.On("GetBucktValue", fileModel.Path).Return(nil, nil)
	mockSetUp.cacheManager.On("SetBucktValue", fileModel.Path, []byte("file data")).Return(nil)

	file, err := mockSetUp.fileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFile_CacheMiss_RepoMiss(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(nil, nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(nil, fmt.Errorf("file not found"))

	file, err := mockSetUp.fileService.GetFile(fileID.String())
	assert.Error(t, err)
	assert.Nil(t, file)
}

func TestGetFile_CacheHit_FileDataCacheHit(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	jsonData, _ := json.Marshal(fileModel)

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(string(jsonData), nil)

	mockSetUp.backend.On("Get", fileModel.Path).Return([]byte("file data"), nil)

	file, err := mockSetUp.fileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFile_CacheHit_FileDataCacheMiss(t *testing.T) {
	mockSetUp := setupFileTest()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	jsonData, _ := json.Marshal(fileModel)

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(string(jsonData), nil)
	mockSetUp.cacheManager.On("GetBucktValue", fileModel.Path).Return(nil, nil)
	mockSetUp.backend.On("Get", fileModel.Path).Return([]byte("file data"), nil)
	mockSetUp.cacheManager.On("SetBucktValue", fileModel.Path, []byte("file data")).Return(nil)

	file, err := mockSetUp.fileService.GetFile(fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}
