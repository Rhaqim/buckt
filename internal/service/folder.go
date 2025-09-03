package service

import (
	"encoding/json"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/constant"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
)

type FolderService struct {
	logger domain.BucktLogger

	cache domain.CacheManager

	repo domain.FolderRepository

	backend domain.FileBackend
}

func NewFolderService(
	bucktLogger domain.BucktLogger,
	cacheManager domain.CacheManager,
	folderRepository domain.FolderRepository,
	backend domain.FileBackend,
) domain.FolderService {
	bucktLogger.Info("🚀 Initialising folder services")
	return &FolderService{
		logger:  bucktLogger,
		cache:   cacheManager,
		repo:    folderRepository,
		backend: backend,
	}
}

// CreateFolder implements domain.FolderService.
func (f *FolderService) CreateFolder(user_id, parent_id, folder_name, description string) (string, error) {
	var err error
	var parentFolder *model.FolderModel

	if parent_id == "" {
		parent_id = constant.DEFAULT_PARENT_FOLDER_ID
	}

	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return "", f.logger.WrapError("failed to parse uuid", err)
	}
	// Get the parent folder
	parentFolder, err = f.repo.GetFolder(parentID)
	if err != nil {
		parentFolder, err = f.GetRootFolder(user_id)
		if err != nil {
			return "", err
		}
	}

	path := filepath.Join(parentFolder.Path, folder_name)

	folder := &model.FolderModel{
		UserID:      user_id,
		ParentID:    &parentFolder.ID,
		Name:        folder_name,
		Description: description,
		Path:        path,
	}

	new_folder_id, err := f.repo.Create(folder)
	if err != nil {
		return "", f.logger.WrapError("failed to create folder", err)
	}

	return new_folder_id, nil
}

// GetFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolder of FolderService.repo.
func (f *FolderService) GetFolder(user_id, folder_id string) (*model.FolderModel, error) {
	if folder_id == "" {
		folder_id = constant.DEFAULT_PARENT_FOLDER_ID
	}

	id, err := uuid.Parse(folder_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to parse uuid", err)
	}

	// Cache key format
	cacheKey := "folder:" + folder_id

	// Try retrieving from cache
	if f.cache != nil {
		cached, err := f.cache.GetBucktValue(cacheKey)
		// Check if cached data is present and valid
		if err == nil && cached != nil {
			cachedStr, ok := cached.(string)
			if ok {
				var cachedFolder model.FolderModel
				if jsonErr := json.Unmarshal([]byte(cachedStr), &cachedFolder); jsonErr == nil {
					return &cachedFolder, nil
				}
			}
		}
	}

	// If not found in cache, fetch from database
	folderPtr, err := f.repo.GetFolder(id)
	if err != nil {
		if err.Error() == "record not found" {
			folderPtr, err = f.repo.GetRootFolder(user_id)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, f.logger.WrapError("failed to get folder", err)
		}
	}

	// Store in cache before returning
	if f.cache != nil {
		if jsonBytes, jsonErr := json.Marshal(folderPtr); jsonErr == nil {
			_ = f.cache.SetBucktValue(cacheKey, string(jsonBytes)) // Ignore cache error
		}
	}

	return folderPtr, nil
}

// GetRootFolder implements domain.FolderService.
func (f *FolderService) GetRootFolder(user_id string) (*model.FolderModel, error) {

	rootFolder, err := f.repo.GetRootFolder(user_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to get root folder", err)
	}

	return rootFolder, nil
}

// GetFolders implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolders of FolderService.repo.
func (f *FolderService) GetFolders(parent_id string) ([]model.FolderModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to parse uuid", err)
	}

	folders, err := f.repo.GetFolders(parentID)
	if err != nil {
		return nil, f.logger.WrapError("failed to get folders", err)
	}

	return folders, nil
}

// MoveFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).MoveFolder of FolderService.repo.
func (f *FolderService) MoveFolder(folder_id string, new_parent_id string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	newParentID, err := uuid.Parse(new_parent_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	if err := f.repo.MoveFolder(folderID, newParentID); err != nil {
		return f.logger.WrapError("failed to move folder", err)
	}

	return nil
}

// RenameFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).RenameFolder of FolderService.repo.
func (f *FolderService) RenameFolder(user_id string, folder_id string, new_name string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	if err := f.repo.RenameFolder(user_id, folderID, new_name); err != nil {
		return f.logger.WrapError("failed to rename folder", err)
	}

	return nil
}

// DeleteFolder implements domain.FolderService.
func (f *FolderService) DeleteFolder(folder_id string) (string, error) {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return "", f.logger.WrapError("failed to parse uuid", err)
	}

	parent_id, err := f.repo.DeleteFolder(folderID)
	if err != nil {
		return "", f.logger.WrapError("failed to delete folder", err)
	}

	return parent_id, nil
}

// ScrubFolder implements domain.FolderService.
func (f *FolderService) ScrubFolder(user_id, folder_id string) (string, error) {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return "", f.logger.WrapError("failed to parse uuid", err)
	}

	// get folder
	folder, err := f.repo.GetFolder(folderID)
	if err != nil {
		return "", f.logger.WrapError("failed to get folder", err)
	}

	err = f.backend.DeleteFolder(folder.Path)
	if err != nil {
		return "", f.logger.WrapError("failed to delete folder", err)
	}

	parent_id, err := f.repo.ScrubFolder(user_id, folderID)
	if err != nil {
		return "", f.logger.WrapError("failed to scrub folder", err)
	}

	return parent_id, nil
}
