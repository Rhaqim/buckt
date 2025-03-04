package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type FileService struct {
	*logger.BucktLogger

	domain.CacheManager

	flatNameSpaces bool

	domain.FileRepository

	domain.FolderService
	domain.FileSystemService
}

func NewFileService(
	bucktLogger *logger.BucktLogger,

	cacheManager domain.CacheManager,

	flatNameSpaces bool,

	fileRepository domain.FileRepository,

	folderService domain.FolderService,
	fileSystemService domain.FileSystemService,
) domain.FileService {
	return &FileService{
		BucktLogger: bucktLogger,

		CacheManager: cacheManager,

		flatNameSpaces: flatNameSpaces,

		FileRepository: fileRepository,

		FolderService:     folderService,
		FileSystemService: fileSystemService,
	}
}

// CreateFile implements domain.FileService.
func (f *FileService) CreateFile(user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	var err error

	// Get the parent folder
	parentFolder, err := f.FolderService.GetFolder("user1", parent_id)
	if err != nil {
		parentFolder, err = f.FolderService.GetRootFolder(user_id)
		if err != nil {
			return "", err
		}
	}

	// Get the file path
	path := filepath.Join(parentFolder.Path, file_name)

	// if flat namespaces is enabled save files in the root folder with uuid as name
	if f.flatNameSpaces {
		ext := filepath.Ext(file_name)
		path = uuid.New().String() + ext
	}

	// Calculate the file hash, for data verification
	combinedData := append([]byte(path), file_data...)
	hash := fmt.Sprintf("%x", sha256.Sum256(combinedData))

	// Size of the file
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
		return "", err
	}

	// Create the file
	err = f.FileRepository.Create(file)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: file_models.hash" {
			file, err = f.FileRepository.RestoreFile(hash)
			if err != nil {
				return "", f.WrapError("failed to restore file", err)
			}
		} else {
			return "", f.WrapError("failed to create file", err)
		}
	}

	return file.ID.String(), nil
}

// GetFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFile of FileService.FileRepository.
func (f *FileService) GetFile(file_id string) (*model.FileModel, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, f.WrapError("failed to parse uuid", err)
	}

	var file *model.FileModel

	// Check cache first
	if f.CacheManager != nil {
		cached, err := f.CacheManager.Get(file_id)
		if err == nil {
			var cachedFile model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFile); jsonErr == nil {
				file = &cachedFile
			}
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.FileRepository.GetFile(fileID)
		if err != nil {
			return nil, f.WrapError("failed to get file metadata", err)
		}

		// Store metadata in cache (without file data)
		if f.CacheManager != nil {
			jsonData, _ := json.Marshal(file) // Ignore errors for now
			_ = f.CacheManager.Set(file_id, string(jsonData))
		}
	}

	// Fetch actual file data separately
	fileData, err := f.FileSystemService.FSGetFile(file.Path)
	if err != nil {
		return nil, f.WrapError("failed to get file data", err)
	}

	// Attach file data
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

	var files []*model.FileModel

	// Generate cache key
	cacheKey := fmt.Sprintf("files:%s", parent_id)

	// Check cache first
	if f.CacheManager != nil {
		cached, err := f.CacheManager.Get(cacheKey)
		if err == nil {
			var cachedFiles []*model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFiles); jsonErr == nil {
				files = cachedFiles
			}
		}
	}

	// If not found in cache, fetch from repository
	if len(files) == 0 {
		files, err = f.FileRepository.GetFiles(parentID)
		if err != nil {
			return nil, f.WrapError("failed to get files", err)
		}

		// Store metadata in cache (without file data)
		if f.CacheManager != nil {
			jsonData, _ := json.Marshal(files) // Ignore errors for now
			_ = f.CacheManager.Set(cacheKey, string(jsonData))
		}
	}

	// Fetch actual file data separately
	var fileModels []model.FileModel
	for _, file := range files {
		fileData, err := f.FileSystemService.FSGetFile(file.Path)
		if err != nil {
			return nil, f.WrapError("failed to get file data", err)
		}
		file.Data = fileData
		fileModels = append(fileModels, *file)
	}

	return fileModels, nil
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
	parentFolder, err := f.FolderService.GetFolder("user1", file.ParentID.String())
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
func (f *FileService) DeleteFile(file_id string) (string, error) {
	var parentID string

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return parentID, f.WrapError("failed to parse uuid", err)
	}

	file, err := f.FileRepository.GetFile(fileID)
	if err != nil {
		return parentID, f.WrapError("failed to get file", err)
	}

	// Delete the file from the file system
	if err := f.FileSystemService.FSDeleteFile(file.Path); err != nil {
		return parentID, err
	}

	// Delete the file
	if err := f.FileRepository.DeleteFile(fileID); err != nil {
		return parentID, f.WrapError("failed to delete file", err)
	}

	return file.ParentID.String(), nil
}
