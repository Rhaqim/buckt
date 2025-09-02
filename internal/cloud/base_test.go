package cloud

import (
	"context"
	"errors"
	"testing"

	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBaseCloudStorage_UploadFileToCloud(t *testing.T) {
	mockFileService := new(mocks.FileService)
	mockFile := &model.FileModel{
		Name:        "test.txt",
		ContentType: "text/plain",
		Path:        "/test.txt",
		Data:        []byte("test data"),
	}
	mockFileService.On("GetFile", "file_id").Return(mockFile, nil)

	uploadFileFn := func(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
		return nil
	}

	baseCloudStorage := &BaseCloudStorage{
		Ctx:          context.Background(),
		FileService:  mockFileService,
		UploadFileFn: uploadFileFn,
	}

	err := baseCloudStorage.UploadFileToCloud("file_id")
	assert.NoError(t, err)
	mockFileService.AssertExpectations(t)
}

func TestBaseCloudStorage_UploadFolderToCloud(t *testing.T) {
	mockFolderService := new(mocks.FolderService)
	mockFileService := new(mocks.FileService) // Add mock file service

	mockFolder := &model.FolderModel{
		ID:   uuid.New(),
		Path: "/folder",
		Files: []model.FileModel{
			{
				Name:        "test.txt",
				ContentType: "text/plain",
				Path:        "/folder/test.txt",
				Data:        []byte("test data"),
			},
		},
		Folders: []model.FolderModel{},
	}

	mockFolderService.On("GetFolder", "user_id", "folder_id").Return(mockFolder, nil)

	uploadFileFn := func(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
		return nil
	}

	createEmptyFolderFn := func(folderPath string) error {
		return nil
	}

	baseCloudStorage := &BaseCloudStorage{
		Ctx:                 context.Background(),
		FolderService:       mockFolderService,
		FileService:         mockFileService, // Set the mock FileService
		UploadFileFn:        uploadFileFn,
		CreateEmptyFolderFn: createEmptyFolderFn,
	}

	err := baseCloudStorage.UploadFolderToCloud("user_id", "folder_id")
	assert.NoError(t, err)

	mockFolderService.AssertExpectations(t)
	mockFileService.AssertExpectations(t) // Ensure the expectations for FileService are met
}

func TestBaseCloudStorage_UploadFileToCloud_Error(t *testing.T) {
	mockFileService := new(mocks.FileService)

	baseCloudStorage := &BaseCloudStorage{
		Ctx:         context.Background(),
		FileService: mockFileService,
	}

	mockFileService.On("GetFile", "file_id").Return(nil, errors.New("file not found"))

	err := baseCloudStorage.UploadFileToCloud("file_id")
	assert.Error(t, err)
	assert.Equal(t, "file not found", err.Error())
	mockFileService.AssertExpectations(t)
}

func TestBaseCloudStorage_UploadFolderToCloud_Error(t *testing.T) {
	mockFolderService := new(mocks.FolderService)
	mockFolderService.On("GetFolder", "user_id", "folder_id").Return(nil, errors.New("folder not found"))

	baseCloudStorage := &BaseCloudStorage{
		Ctx:           context.Background(),
		FolderService: mockFolderService,
	}

	err := baseCloudStorage.UploadFolderToCloud("user_id", "folder_id")
	assert.Error(t, err)
	assert.Equal(t, "folder not found", err.Error())
	mockFolderService.AssertExpectations(t)
}
