package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderRepository struct {
	db *database.DB
}

func NewFolderRepository(db *database.DB) domain.FolderRepository {
	return &FolderRepository{db: db}
}

// Create implements domain.FolderRepository.
// Subtle: this method shadows the method (*DB).Create of FolderRepository.DB.WithContext(ctx).
func (f *FolderRepository) Create(ctx context.Context, folder *model.FolderModel) (string, error) {
	if err := f.db.DB.WithContext(ctx).Create(folder).Error; err != nil {
		return "", err
	}
	return folder.ID.String(), nil
}

// GetFolder implements domain.FolderRepository.
func (f *FolderRepository) GetFolder(ctx context.Context, folder_id uuid.UUID) (*model.FolderModel, error) {
	var folder model.FolderModel
	err := f.db.DB.WithContext(ctx).Preload("Folders").Preload("Files").Where("id = ?", folder_id).First(&folder).Error
	return &folder, err
}

// GetRootFolder implements domain.FolderRepository.
// look for a folder called root and return it, if root doesn't exist, create it
func (f *FolderRepository) GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	root := model.FolderModel{}

	root_folder := "root_folder"

	err := f.db.DB.WithContext(ctx).Preload("Folders").Preload("Files").Where("name = ? AND user_id = ?", root_folder, user_id).First(&root).Error
	if err != nil {
		if err.Error() != "record not found" {
			return nil, err
		}

		path := "/" + user_id + "/" + root_folder

		if err := f.db.DB.WithContext(ctx).Create(&model.FolderModel{
			UserID:      user_id,
			Name:        root_folder,
			Description: "Root folder",
			Path:        path,
		}).Error; err != nil {
			return nil, err
		}

		return f.GetRootFolder(ctx, user_id)
	}

	return &root, nil
}

// GetFolders implements domain.FolderRepository.
func (f *FolderRepository) GetFolders(ctx context.Context, parent_id uuid.UUID) ([]model.FolderModel, error) {
	var folders []model.FolderModel
	err := f.db.DB.WithContext(ctx).Where("parent_id = ?", parent_id).Find(&folders).Error
	return folders, err
}

// MoveFolder implements domain.FolderRepository.
func (f *FolderRepository) MoveFolder(ctx context.Context, folder_id uuid.UUID, new_parent_id uuid.UUID) error {
	var newParentFolder model.FolderModel

	if err := f.db.DB.WithContext(ctx).Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("parent folder not found")
		}
		return err
	}

	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ?", folder_id).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("folder not found")
		}
		return err
	}

	// Prevent moving into its own subfolder
	if strings.HasPrefix(newParentFolder.Path, folder.Path) {
		return fmt.Errorf("invalid move: cannot move a folder into its own subfolder")
	}

	// Construct new path safely
	newPath := strings.TrimSuffix(newParentFolder.Path, "/") + "/" + folder.Name

	// Avoid unnecessary updates
	if folder.Path == newPath && folder.ParentID == &newParentFolder.ID {
		return nil
	}

	// Update both `path` and `parent_id`
	return f.db.DB.WithContext(ctx).Model(&folder).Updates(map[string]interface{}{
		"path":      newPath,
		"parent_id": newParentFolder.ID,
	}).Error

}

// RenameFolder implements domain.FolderRepository.
func (f *FolderRepository) RenameFolder(ctx context.Context, user_id string, folder_id uuid.UUID, new_name string) error {
	// get the folder to rename
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("folder not found")
		}
		return err
	}

	// update the folder name and path
	newPath := strings.TrimSuffix(folder.Path, "/"+folder.Name) + "/" + new_name
	return f.db.DB.WithContext(ctx).Model(&folder).Updates(map[string]interface{}{
		"name": new_name,
		"path": newPath,
	}).Error
}

// DeleteFolder implements domain.FolderRepository.
func (f *FolderRepository) DeleteFolder(ctx context.Context, folder_id uuid.UUID) (parent_id string, err error) {
	folder, err := f.GetFolder(ctx, folder_id)
	if err != nil {
		return "", err
	}

	// delete the folder
	if err := f.db.DB.WithContext(ctx).Delete(&folder).Error; err != nil {
		return "", err
	}

	return folder.ParentID.String(), nil
}

// ScrubFolder implements domain.FolderRepository.
func (f *FolderRepository) ScrubFolder(ctx context.Context, user_id string, folder_id uuid.UUID) (parent_id string, err error) {
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("folder not found")
		}
		return "", err
	}

	// delete the folder
	if err := f.db.DB.WithContext(ctx).Unscoped().Delete(&folder).Error; err != nil {
		return "", err
	}

	return folder.ParentID.String(), nil
}
