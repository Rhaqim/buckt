package cloud

import (
	"context"
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type uploadFileFn func(file_name, content_type, file_path string, data []byte, metadata map[string]string) error

// BaseCloudStorage provides common logic for all cloud providers.
type BaseCloudStorage struct {
	ctx                 context.Context
	FileService         domain.FileService
	FolderService       domain.FolderService
	UploadFileFn        uploadFileFn
	CreateEmptyFolderFn func(folderPath string) error
}

// UploadFolder implements domain.CloudService.
func (a *BaseCloudStorage) UploadFolderToCloud(user_id, folder_id string) error {
	// get the folder with content
	folder, err := a.FolderService.GetFolder(user_id, folder_id)
	if err != nil {
		return err
	}

	// upload all files in the folder
	if err := a.recursiveUpload(*folder, make(map[string]bool)); err != nil {
		return err
	}

	return nil
}

// UploadFile implements domain.CloudService.
func (a *BaseCloudStorage) UploadFileToCloud(file_id string) error {
	// Get the file from the file service.
	file, err := a.FileService.GetFile(file_id)
	if err != nil {
		return err
	}

	if file == nil {
		return fmt.Errorf("file not found")
	}

	if file.Data == nil {
		return fmt.Errorf("file data is empty")
	}

	if a.UploadFileFn == nil {
		return fmt.Errorf("upload file function is not set")
	}

	// Set metadata for the file.
	metadata := make(map[string]string) // ✅ Ensure map is initialized

	if file.ID != uuid.Nil {
		metadata["file_id"] = file.ID.String()
	}

	if file.Hash != "" {
		metadata["file_hash"] = file.Hash
	}

	// Upload the file to the cloud.
	if err := a.UploadFileFn(file.Name, file.ContentType, file.Path, file.Data, metadata); err != nil {
		return err
	}

	return nil
}

func (b *BaseCloudStorage) recursiveUpload(folder model.FolderModel, visited map[string]bool) error {
	if visited[folder.ID.String()] {
		return fmt.Errorf("circular reference detected in folder structure")
	}
	visited[folder.ID.String()] = true

	if b.CreateEmptyFolderFn == nil {
		return fmt.Errorf("create empty folder function is not set")
	}

	// Ensure empty folder exists
	if err := b.CreateEmptyFolderFn(folder.Path + "/"); err != nil {
		return err
	}

	// Upload files
	for _, file := range folder.Files {
		metadata := make(map[string]string) // ✅ Ensure map is initialized

		if file.ID != uuid.Nil {
			metadata["file_id"] = file.ID.String()
		}

		if file.Hash != "" {
			metadata["file_hash"] = file.Hash
		}

		if err := b.UploadFileFn(file.Name, file.ContentType, file.Path, file.Data, metadata); err != nil {
			return err
		}
	}

	// Recursively upload subfolders
	for _, subFolder := range folder.Folders {
		if err := b.recursiveUpload(subFolder, visited); err != nil {
			return err
		}
	}
	return nil
}
