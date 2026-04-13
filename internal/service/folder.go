package service

import (
	"context"
	"encoding/json"
	"fmt"
	"errors"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/constant"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderService struct {
	logger domain.BucktLogger

	cache domain.CacheManager

	repo domain.FolderRepository

	backend        domain.FileBackend
	flatNameSpaces bool
}

func NewFolderService(
	bucktLogger domain.BucktLogger,
	cacheManager domain.CacheManager,
	folderRepository domain.FolderRepository,
	backend domain.FileBackend,
	flatNameSpaces bool,
) domain.FolderService {
	bucktLogger.Info("🚀 Initialising folder services")
	return &FolderService{
		logger:         bucktLogger,
		cache:          cacheManager,
		repo:           folderRepository,
		backend:        backend,
		flatNameSpaces: flatNameSpaces,
	}
}

// CreateFolder implements domain.FolderService.
func (f *FolderService) CreateFolder(ctx context.Context, user_id, parent_id, folder_name, description string) (string, error) {
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
	parentFolder, err = f.repo.GetFolder(ctx, parentID)
	if err != nil {
		parentFolder, err = f.GetRootFolder(ctx, user_id)
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

	new_folder_id, err := f.repo.Create(ctx, folder)
	if err != nil {
		return "", f.logger.WrapError("failed to create folder", err)
	}

	return new_folder_id, nil
}

// GetFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolder of FolderService.repo.
func (f *FolderService) GetFolder(ctx context.Context, user_id, folder_id string) (*model.FolderModel, error) {
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
		cached, err := f.cache.GetBucktValue(ctx, cacheKey)
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
	folderPtr, err := f.repo.GetFolder(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			folderPtr, err = f.repo.GetRootFolder(ctx, user_id)
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
			_ = f.cache.SetBucktValue(ctx, cacheKey, string(jsonBytes)) // Ignore cache error
		}
	}

	return folderPtr, nil
}

// GetRootFolder implements domain.FolderService.
func (f *FolderService) GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {

	rootFolder, err := f.repo.GetRootFolder(ctx, user_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to get root folder", err)
	}

	return rootFolder, nil
}

// GetTrashFolder implements domain.FolderService.
// Returns the user's trash folder with its current contents preloaded.
func (f *FolderService) GetTrashFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	trash, err := f.repo.GetTrashFolder(ctx, user_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to get trash folder", err)
	}

	// Preload contents
	return f.repo.GetFolder(ctx, trash.ID)
}

// GetFolders implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolders of FolderService.repo.
func (f *FolderService) GetFolders(ctx context.Context, parent_id string) ([]model.FolderModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, f.logger.WrapError("failed to parse uuid", err)
	}

	folders, err := f.repo.GetFolders(ctx, parentID)
	if err != nil {
		return nil, f.logger.WrapError("failed to get folders", err)
	}

	return folders, nil
}

// MoveFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).MoveFolder of FolderService.repo.
func (f *FolderService) MoveFolder(ctx context.Context, folder_id string, new_parent_id string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	newParentID, err := uuid.Parse(new_parent_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	if err := f.repo.MoveFolder(ctx, folderID, newParentID); err != nil {
		return f.logger.WrapError("failed to move folder", err)
	}

	return nil
}

// RenameFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).RenameFolder of FolderService.repo.
func (f *FolderService) RenameFolder(ctx context.Context, user_id string, folder_id string, new_name string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.logger.WrapError("failed to parse uuid", err)
	}

	if err := f.repo.RenameFolder(ctx, user_id, folderID, new_name); err != nil {
		return f.logger.WrapError("failed to rename folder", err)
	}

	return nil
}

// DeleteFolder implements domain.FolderService.
// Moves the folder to trash (or hard-deletes if already in trash) and
// coordinates the storage backend so files physically follow their new paths.
// Backend operations run inside the DB transaction so a backend failure
// rolls back the path rewrite, keeping the DB and backend consistent.
func (f *FolderService) DeleteFolder(ctx context.Context, folder_id string) (string, error) {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return "", f.logger.WrapError("failed to parse uuid", err)
	}

	parent_id, _, _, _, err := f.repo.DeleteFolder(ctx, folderID, func(oldPath, newPath string, fileMoves []model.PathMove) error {
		// In flat-namespace mode the paths don't reflect disk layout, so no
		// backend op is needed.
		if f.flatNameSpaces {
			return nil
		}

		if newPath == "" {
			// Permanent deletion — remove the whole prefix from backend
			return f.backend.DeleteFolder(ctx, oldPath)
		}

		// Move every file under the folder to its new path. If any move
		// fails, returning an error rolls back the entire DB transaction.
		for _, mv := range fileMoves {
			if err := f.backend.Move(ctx, mv.Old, mv.New); err != nil {
				return fmt.Errorf("failed to move %s to %s: %w", mv.Old, mv.New, err)
			}
		}
		return nil
	})
	if err != nil {
		return "", f.logger.WrapError("failed to delete folder", err)
	}

	return parent_id, nil
}

// ScrubFolder implements domain.FolderService.
func (f *FolderService) ScrubFolder(ctx context.Context, user_id, folder_id string) (string, error) {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return "", f.logger.WrapError("failed to parse uuid", err)
	}

	// get folder
	folder, err := f.repo.GetFolder(ctx, folderID)
	if err != nil {
		return "", f.logger.WrapError("failed to get folder", err)
	}

	err = f.backend.DeleteFolder(ctx, folder.Path)
	if err != nil {
		return "", f.logger.WrapError("failed to delete folder", err)
	}

	parent_id, err := f.repo.ScrubFolder(ctx, user_id, folderID)
	if err != nil {
		return "", f.logger.WrapError("failed to scrub folder", err)
	}

	return parent_id, nil
}
