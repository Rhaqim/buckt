package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/Rhaqim/buckt/pkg/scan"
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

	fileService := NewFileService(mockLogger, mockCache, mockFileRepo, mockFolderService, mockLocalFileSystemService, false, 100*1024*1024, 0, nil, false, nil)

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
	ctx := t.Context()

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

	_, err := mockSetUp.fileService.CreateFile(ctx, user_id, "parent_id", "file.txt", "text/plain", []byte("file data"))
	assert.NoError(t, err)
}

// TestCreateFile_BackendPutFailureRollsBackMetadata verifies that when the
// backend write fails after the metadata row was created, the service
// compensates by scrubbing the row (so no FileModel points at a missing blob)
// and returns the wrapped backend error.
func TestCreateFile_BackendPutFailureRollsBackMetadata(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	parentFolder := &model.FolderModel{ID: uuid.New(), Path: "/parent/folder"}
	user_id := "user1"

	mockSetUp.folderService.On("GetFolder", user_id, "parent_id").Return(parentFolder, nil)
	mockSetUp.fileRepository.On("Create", mock.Anything).Return(nil)

	putErr := fmt.Errorf("backend unavailable")
	mockSetUp.backend.On("Put", "/parent/folder/file.txt", []byte("data")).Return(putErr)

	// The compensation must delete the just-created row.
	mockSetUp.fileRepository.On("ScrubFile", mock.Anything).Return(nil)

	_, err := mockSetUp.fileService.CreateFile(ctx, user_id, "parent_id", "file.txt", "text/plain", []byte("data"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, putErr)
	mockSetUp.fileRepository.AssertCalled(t, "ScrubFile", mock.Anything)
}

// setupFileTestWithScanner builds a file service wired with the given scanner so
// the upload-rejection path can be exercised.
func setupFileTestWithScanner(s scan.Scanner) MockFileServices {
	mockLogger := logger.NewLogger("", true, false)
	mockCache := new(mocks.CacheManager)
	mockFileRepo := new(mocks.FileRepository)
	mockFolderService := new(mocks.FolderService)
	mockBackend := new(mocks.LocalFileSystemService)

	fileService := NewFileService(mockLogger, mockCache, mockFileRepo, mockFolderService, mockBackend, false, 100*1024*1024, 0, nil, false, s)

	return MockFileServices{
		fileService:    fileService,
		cacheManager:   mockCache,
		fileRepository: mockFileRepo,
		folderService:  mockFolderService,
		backend:        mockBackend,
	}
}

// TestCreateFile_ScannerRejectsUpload verifies that when the scanner returns an
// error, the upload is rejected as ErrUploadRejected before any DB row or blob
// is written, and the scanner's own error remains inspectable.
func TestCreateFile_ScannerRejectsUpload(t *testing.T) {
	scanErr := fmt.Errorf("eicar signature detected")
	rejecting := scan.ScannerFunc(func(_ context.Context, _ string, _ []byte) error {
		return scanErr
	})
	mockSetUp := setupFileTestWithScanner(rejecting)
	ctx := t.Context()

	_, err := mockSetUp.fileService.CreateFile(ctx, "user1", "parent_id", "virus.exe", "application/octet-stream", []byte("malicious"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, buckterr.ErrUploadRejected)
	assert.ErrorIs(t, err, scanErr)

	// Rejection happens before any DB or backend work — none of these run.
	mockSetUp.folderService.AssertNotCalled(t, "GetFolder", mock.Anything, mock.Anything)
	mockSetUp.fileRepository.AssertNotCalled(t, "Create", mock.Anything)
	mockSetUp.backend.AssertNotCalled(t, "Put", mock.Anything, mock.Anything)
}

// TestCreateFile_ScannerAdmitsUpload verifies a scanner that returns nil lets the
// upload proceed to storage unchanged.
func TestCreateFile_ScannerAdmitsUpload(t *testing.T) {
	admitting := scan.ScannerFunc(func(_ context.Context, _ string, _ []byte) error {
		return nil
	})
	mockSetUp := setupFileTestWithScanner(admitting)
	ctx := t.Context()

	parentFolder := &model.FolderModel{ID: uuid.New(), Path: "/parent/folder"}
	mockSetUp.folderService.On("GetFolder", "user1", "parent_id").Return(parentFolder, nil)
	mockSetUp.fileRepository.On("Create", mock.Anything).Return(nil)
	mockSetUp.backend.On("Put", "/parent/folder/file.txt", []byte("clean data")).Return(nil)

	_, err := mockSetUp.fileService.CreateFile(ctx, "user1", "parent_id", "file.txt", "text/plain", []byte("clean data"))
	assert.NoError(t, err)
	mockSetUp.backend.AssertCalled(t, "Put", "/parent/folder/file.txt", []byte("clean data"))
}

func TestGetFiles(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	files, err := mockSetUp.fileService.GetFiles(ctx, parentID.String())
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, []byte("file1 data"), files[0].Data)
	assert.Equal(t, []byte("file2 data"), files[1].Data)
}

func TestUpdateFile(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	err := mockSetUp.fileService.UpdateFile(ctx, user_id, fileID.String(), "new_file.txt", []byte("new file data"))
	assert.NoError(t, err)
}

// TestDeleteFile_PermanentDeleteFromTrash covers the path where the repo
// returns newPath == "" indicating the file was already in trash and got
// hard-deleted. The service should call backend.Delete on oldPath.
func TestDeleteFile_PermanentDeleteFromTrash(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/user/__trash__/file.txt",
	}

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return("", nil)
	mockSetUp.cacheManager.On("DeleteBucktValue", fileID.String()).Return(nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	// Repo indicates permanent delete via newPath == ""
	mockSetUp.fileRepository.On("DeleteFile", fileID).Return("/user/__trash__/file.txt", "", nil)

	// Service should remove the blob from the backend
	mockSetUp.backend.On("Delete", "/user/__trash__/file.txt").Return(nil)

	_, err := mockSetUp.fileService.DeleteFile(ctx, fileID.String())
	assert.NoError(t, err)
	mockSetUp.backend.AssertCalled(t, "Delete", "/user/__trash__/file.txt")
}

// TestDeleteFile_MoveToTrashNested covers the nested-namespace case where
// the repo returns a different newPath. The service should call backend.Move
// to physically relocate the blob.
func TestDeleteFile_MoveToTrashNested(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/user/photos/sunset.jpg",
	}

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return("", nil)
	mockSetUp.cacheManager.On("DeleteBucktValue", fileID.String()).Return(nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	oldPath := "/user/photos/sunset.jpg"
	newPath := "/user/__trash__/sunset.jpg"
	mockSetUp.fileRepository.On("DeleteFile", fileID).Return(oldPath, newPath, nil)

	// Service should physically move the blob on the backend
	mockSetUp.backend.On("Move", oldPath, newPath).Return(nil)

	_, err := mockSetUp.fileService.DeleteFile(ctx, fileID.String())
	assert.NoError(t, err)
	mockSetUp.backend.AssertCalled(t, "Move", oldPath, newPath)
	mockSetUp.backend.AssertNotCalled(t, "Delete", mock.Anything)
}

// TestDeleteFile_MoveToTrashFlatNamespace covers the flat-namespace case
// where oldPath == newPath. The service should NOT call backend.Move
// because the blob's actual location is unchanged.
func TestDeleteFile_MoveToTrashFlatNamespace(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "abc123.jpg",
	}

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return("", nil)
	mockSetUp.cacheManager.On("DeleteBucktValue", fileID.String()).Return(nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(fileModel, nil)

	// Flat mode: repo returns same path for old and new
	mockSetUp.fileRepository.On("DeleteFile", fileID).Return("abc123.jpg", "abc123.jpg", nil)

	_, err := mockSetUp.fileService.DeleteFile(ctx, fileID.String())
	assert.NoError(t, err)

	// Neither Move nor Delete should be called — blob stays at its flat path
	mockSetUp.backend.AssertNotCalled(t, "Move", mock.Anything, mock.Anything)
	mockSetUp.backend.AssertNotCalled(t, "Delete", mock.Anything)
}

func TestScrubFile(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	parentIDStr, err := mockSetUp.fileService.ScrubFile(ctx, fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, parentID.String(), parentIDStr)
}

func TestGetFile_CacheHit(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	file, err := mockSetUp.fileService.GetFile(ctx, fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, fileModel.Path, file.Path)
}

func TestGetFile_CacheMiss_RepoHit(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	file, err := mockSetUp.fileService.GetFile(ctx, fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFile_CacheMiss_RepoMiss(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	fileID := uuid.New()

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(nil, nil)
	mockSetUp.fileRepository.On("GetFile", fileID).Return(nil, fmt.Errorf("file not found"))

	file, err := mockSetUp.fileService.GetFile(ctx, fileID.String())
	assert.Error(t, err)
	assert.Nil(t, file)
}

func TestGetFile_CacheHit_FileDataCacheHit(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	fileID := uuid.New()
	fileModel := &model.FileModel{
		ID:   fileID,
		Path: "/parent/folder/file.txt",
	}
	jsonData, _ := json.Marshal(fileModel)

	mockSetUp.cacheManager.On("GetBucktValue", fileID.String()).Return(string(jsonData), nil)

	mockSetUp.backend.On("Get", fileModel.Path).Return([]byte("file data"), nil)

	file, err := mockSetUp.fileService.GetFile(ctx, fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFile_CacheHit_FileDataCacheMiss(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

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

	file, err := mockSetUp.fileService.GetFile(ctx, fileID.String())
	assert.NoError(t, err)
	assert.Equal(t, fileModel.ID, file.ID)
	assert.Equal(t, []byte("file data"), file.Data)
}

func TestGetFilesMetaData(t *testing.T) {
	mockSetUp := setupFileTest()
	ctx := t.Context()

	parentID := uuid.New()
	fileModels := []*model.FileModel{
		{ID: uuid.New(), Path: "/parent/folder/file1.txt"},
		{ID: uuid.New(), Path: "/parent/folder/file2.txt"},
	}

	var jsonStr string
	mockSetUp.cacheManager.On("GetBucktValue", "files:"+parentID.String()).Return(jsonStr, nil)
	mockSetUp.cacheManager.On("SetBucktValue", "files:"+parentID.String(), mock.Anything).Return(nil)
	mockSetUp.fileRepository.On("GetFiles", parentID).Return(fileModels, nil)

	files, err := mockSetUp.fileService.GetFilesMetadata(ctx, parentID.String())
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, fileModels[0].ID, files[0].ID)
	assert.Equal(t, fileModels[0].Path, files[0].Path)
}
