package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type FileService struct {
	flatNameSpaces bool

	logger domain.BucktLogger

	cache domain.CacheManager
	repo  domain.FileRepository

	folderService domain.FolderService
	fileBackend   domain.FileBackend
}

func NewFileService(
	bucktLogger domain.BucktLogger,

	cache domain.CacheManager,
	fileRepository domain.FileRepository,

	folderService domain.FolderService,
	fileBackend domain.FileBackend,

	flatNameSpaces bool,
) domain.FileService {
	bucktLogger.Info("🚀 Initialising file services")
	return &FileService{
		logger: bucktLogger,

		cache: cache,

		repo: fileRepository,

		folderService: folderService,
		fileBackend:   fileBackend,

		flatNameSpaces: flatNameSpaces,
	}
}

// CreateFile implements domain.FileService.
func (f *FileService) CreateFile(ctx context.Context, user_id, parent_id, file_name, content_type string, file_data []byte) (string, error) {
	var err error

	// Get the parent folder
	parentFolder, err := f.folderService.GetFolder(ctx, user_id, parent_id)
	if err != nil {
		parentFolder, err = f.folderService.GetRootFolder(ctx, user_id)
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

	// Create the file
	err = f.repo.Create(ctx, file)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: file_models.name, file_models.parent_id" {
			file, err = f.repo.RestoreFile(ctx, file.ParentID, file.Name)
			if err != nil {
				return "", f.logger.WrapError("failed to restore file", err)
			}
		} else {
			return "", f.logger.WrapError("failed to create file", err)
		}
	} else {
		// Write the file to the file system
		if err := f.fileBackend.Put(ctx, file.Path, file_data); err != nil {
			return "", err
		}
	}

	return file.ID.String(), nil
}

// GetFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFile of FileService.repo.
func (f *FileService) GetFile(ctx context.Context, file_id string) (*model.FileModel, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to parse uuid", err)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err != nil {
			f.logger.Warn("failed to get file metadata from cache: " + err.Error())
		}

		if cached != nil {
			cachedStr, ok := cached.(string)
			if ok {
				var cachedFile model.FileModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFile); jsonErr == nil {
					file = &cachedFile
				}
			}
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return nil, f.logger.WrapError("failed to get file metadata", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(file) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, file_id, string(jsonData))
		}
	}

	data, err := f.fileBackend.Get(ctx, file.Path)
	if err != nil {
		return nil, f.logger.WrapError("failed to get file data", err)
	}

	file.Data = data

	return file, nil
}

// GetFileStream implements domain.FileService.
// Subtle: this method shadows the method (FileBackend).GetFilStream of FileService.fileBackend.
func (f *FileService) GetFileStream(ctx context.Context, file_id string) (*model.FileModel, io.ReadCloser, error) {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return nil, nil, f.logger.WrapError("failed to parse uuid", err)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil && cached != nil { // Ensure cached value is not nil
			cachedStr, ok := cached.(string)
			if ok { // Ensure type assertion succeeds
				var cachedFile model.FileModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFile); jsonErr == nil {
					file = &cachedFile
				}
			}
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return nil, nil, f.logger.WrapError("failed to get file metadata", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(file) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, file_id, string(jsonData))
		}
	}

	// Fetch actual file data separately
	fileStream, err := f.fileBackend.Stream(ctx, file.Path)
	if err != nil {
		return nil, nil, f.logger.WrapError("failed to get file data", err)
	}

	return file, fileStream, nil
}

// GetFiles implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).GetFiles of FileService.repo.
func (f *FileService) GetFiles(ctx context.Context, parent_id string) ([]model.FileModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to parse uuid", err)
	}

	var files []*model.FileModel

	// Generate cache key
	cacheKey := fmt.Sprintf("files:%s", parent_id)

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, cacheKey)
		if err == nil || cached != nil {
			var cachedFiles []*model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFiles); jsonErr == nil {
				files = cachedFiles
			}
		}
	}

	// If not found in cache, fetch from repository
	if len(files) == 0 {
		files, err = f.repo.GetFiles(ctx, parentID)
		if err != nil {
			return nil, f.logger.WrapError("failed to get files", err)
		}

		// Store metadata in cache (without file data)
		if f.cache != nil {
			jsonData, _ := json.Marshal(files) // Ignore errors for now
			_ = f.cache.SetBucktValue(ctx, cacheKey, string(jsonData))
		}
	}

	// Fetch actual file data separately
	var fileModels []model.FileModel
	for _, file := range files {
		fileData, err := f.fileBackend.Get(ctx, file.Path)
		if err != nil {
			return nil, f.logger.WrapError("failed to get file data", err)
		}
		file.Data = fileData
		fileModels = append(fileModels, *file)
	}

	return fileModels, nil
}

// MoveFile implements domain.FileService.
func (f *FileService) MoveFile(ctx context.Context, file_id string, new_parent_id string) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	newParentID, err := uuid.Parse(new_parent_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	// Move the file
	oldPath, newPath, err := f.repo.MoveFile(ctx, fileID, newParentID)
	if err != nil {
		return f.logger.WrapError("failed to move file", err)
	}

	if !f.flatNameSpaces {
		// Move the file in the file system
		if err := f.fileBackend.Move(ctx, oldPath, newPath); err != nil {
			return f.logger.WrapError("failed to move file", err)
		}
	}

	return nil
}

// RenameFile implements domain.FileService.
func (f *FileService) RenameFile(ctx context.Context, file_id string, new_name string) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	// Rename the file
	if err := f.repo.RenameFile(ctx, fileID, new_name); err != nil {
		return f.logger.WrapError("failed to rename file", err)
	}

	return nil
}

// UpdateFile implements domain.FileService.
func (f *FileService) UpdateFile(ctx context.Context, user_id, file_id string, new_file_name string, new_file_data []byte) error {
	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	file, err := f.repo.GetFile(ctx, fileID)
	if err != nil {
		return f.logger.WrapError("failed to get file", err)
	}

	// Get the parent folder
	parentFolder, err := f.folderService.GetFolder(ctx, user_id, file.ParentID.String())
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
	if err := f.fileBackend.Put(ctx, newPath, new_file_data); err != nil {
		return err
	}

	// Update the file
	if err := f.repo.Update(ctx, file); err != nil {
		return f.logger.WrapError("failed to update file", err)
	}

	return nil
}

// DeleteFile implements domain.FileService.
// Subtle: this method shadows the method (FileRepository).DeleteFile of FileService.repo.
func (f *FileService) DeleteFile(ctx context.Context, file_id string) (string, error) {
	var parentID string

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return parentID, f.logger.WrapError("failed to parse uuid", err)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil {
			var cachedFile model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFile); jsonErr == nil {
				file = &cachedFile
			}

			// Delete from cache
			_ = f.cache.DeleteBucktValue(ctx, file_id)
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return parentID, f.logger.WrapError("failed to get file metadata", err)
		}
	}

	// Delete the file
	if err := f.repo.DeleteFile(ctx, fileID); err != nil {
		return parentID, f.logger.WrapError("failed to delete file", err)
	}

	return file.ParentID.String(), nil
}

func (f *FileService) ScrubFile(ctx context.Context, file_id string) (string, error) {
	var parentID string

	fileID, err := uuid.Parse(file_id)
	if err != nil {
		return parentID, f.logger.WrapError("failed to parse uuid", err)
	}

	var file *model.FileModel

	// Check cache first
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(ctx, file_id)
		if err == nil {
			var cachedFile model.FileModel
			if jsonErr := json.Unmarshal([]byte(cached.(string)), &cachedFile); jsonErr == nil {
				file = &cachedFile
			}

			// Delete from cache
			_ = f.cache.DeleteBucktValue(ctx, file_id)
		}
	}

	// If not found in cache, fetch from repository
	if file == nil {
		file, err = f.repo.GetFile(ctx, fileID)
		if err != nil {
			return parentID, f.logger.WrapError("failed to get file metadata", err)
		}
	}

	// Delete the file from the file system
	if err := f.fileBackend.Delete(ctx, file.Path); err != nil {
		return parentID, err
	}

	// Delete the file
	if err := f.repo.ScrubFile(ctx, fileID); err != nil {
		return parentID, f.logger.WrapError("failed to delete file", err)
	}

	return file.ParentID.String(), nil
}
