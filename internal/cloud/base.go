package cloud

import (
	"context"
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
)

// BaseCloudStorage provides common logic for all cloud providers.
type BaseCloudStorage struct {
	ctx                 context.Context
	FileService         domain.FileService
	FolderService       domain.FolderService
	UploadFileFn        func(file model.FileModel) error
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
	if err := a.recursiveUpload(*folder, nil); err != nil {
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

	// Upload the file to the cloud.
	if err := a.UploadFileFn(*file); err != nil {
		return err
	}

	return nil
}

func (b *BaseCloudStorage) recursiveUpload(folder model.FolderModel, visited map[string]bool) error {
	if visited[folder.ID.String()] {
		return fmt.Errorf("circular reference detected in folder structure")
	}
	visited[folder.ID.String()] = true

	// Ensure empty folder exists
	if err := b.CreateEmptyFolderFn(folder.Path + "/"); err != nil {
		return err
	}

	// Upload files
	for _, file := range folder.Files {
		if err := b.UploadFileFn(file); err != nil {
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
