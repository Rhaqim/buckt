package service

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type FileService struct {
	*logger.Logger

	domain.FileRepository

	domain.FolderService
	domain.FileSystemService
}

func NewFileService(
	log *logger.Logger,
	fileRepository domain.FileRepository,
	folderService domain.FolderService,
	fileSystemService domain.FileSystemService,
) domain.FileService {
	return &FileService{
		Logger:            log,
		FileRepository:    fileRepository,
		FolderService:     folderService,
		FileSystemService: fileSystemService,
	}
}

// CreateFile implements domain.FileService.
func (f *FileService) CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) error {
	// Get the parent folder
	parentFolder, err := f.FolderService.GetFolder(parent_id)
	if err != nil {
		parentFolder, err = f.FolderService.GetRootFolder(user_id)
		if err != nil {
			return err
		}
	}

	// Get the file path
	// path := parentFolder.Path + "/" + file_name

	path := filepath.Join(parentFolder.Path, file_name)

	// Calculate the file hash, for data verification
	hash := fmt.Sprintf("%x", sha256.Sum256(file_data))

	// Size of the file
	// File size in bytes
	fileSize := int64(len(file_data))

	// Create the file model
	file := &model.FileModel{
		ParentID:    parentFolder.ID,
		Name:        file_name,
		Path:        path,
		Hash:        hash,
		ContentType: content_type,
		Size:        fileSize,
	}

	// Write the file to the file system
	if err := f.FileSystemService.FSWriteFile(path, file_data); err != nil {
		return err
	}

	// check if the file already exists
	if err := f.FileRepository.RestoreFile(hash); err != nil {
		if err.Error() != "record not found" {
			// Create the file
			if err := f.FileRepository.Create(file); err != nil {
				return f.WrapError("failed to create file", err)
			}
		} else {
			// Update the file
			if err := f.FileRepository.Update(file); err != nil {
				return f.WrapError("failed to update file", err)
			}
		}
	}

	return nil
}

// GetFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFile of FileService.FileRepository.
func (f *FileService) GetFile(file_id string) (*model.FileModel, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, f.WrapError("failed to parse uuid", err)
	}

	file, err := f.FileRepository.GetFile(fileID)
	if err != nil {
		return nil, f.WrapError("failed to get file", err)
	}

	// Get the file data
	fileData, err := f.FileSystemService.FSGetFile(file.Path)
	if err != nil {
		return nil, err
	}

	// Create the file model
	file.Data = fileData

	return file, nil
}

// GetFiles implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFiles of FileService.FileRepository.
func (f *FileService) GetFiles(parent_id string) ([]model.FileModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, f.WrapError("failed to parse uuid", err)
	}

	files, err := f.FileRepository.GetFiles(parentID)
	if err != nil {
		return nil, f.WrapError("failed to get files", err)
	}

	// Create the file models
	for _, file := range files {
		// Get the file data
		fileData, err := f.FileSystemService.FSGetFile(file.Path)
		if err != nil {
			return nil, err
		}

		file.Data = fileData
	}

	return files, nil
}

// UpdateFile implements domain.FileService.
func (f *FileService) UpdateFile(file_id string, new_file_name string, new_file_data []byte) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}

	file, err := f.FileRepository.GetFile(fileID)
	if err != nil {
		return f.WrapError("failed to get file", err)
	}

	// Get the parent folder
	parentFolder, err := f.FolderService.GetFolder(file.ParentID.String())
	if err != nil {
		return err
	}

	// Get the new file path
	newPath := parentFolder.Path + "/" + new_file_name

	// Calculate the new file hash, for data verification
	newHash := fmt.Sprintf("%x", sha256.Sum256(new_file_data))

	// Update the file model
	file.Name = new_file_name
	file.Path = newPath
	file.Hash = newHash

	// Update the file in the file system
	if err := f.FileSystemService.FSWriteFile(newPath, new_file_data); err != nil {
		return err
	}

	// Update the file
	if err := f.FileRepository.Update(file); err != nil {
		return f.WrapError("failed to update file", err)
	}

	return nil
}

// DeleteFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).DeleteFile of FileService.FileRepository.
func (f *FileService) DeleteFile(file_id string) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}

	file, err := f.FileRepository.GetFile(fileID)
	if err != nil {
		return f.WrapError("failed to get file", err)
	}

	// Delete the file from the file system
	if err := f.FileSystemService.FSDeleteFile(file.Path); err != nil {
		return err
	}

	// Delete the file
	if err := f.FileRepository.DeleteFile(fileID); err != nil {
		return f.WrapError("failed to delete file", err)
	}

	return nil
}
