package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rhaqim/buckt/internal/constant"
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
func (f *FolderRepository) Create(ctx context.Context, folder *model.FolderModel) (string, error) {
	if err := f.db.DB.WithContext(ctx).Create(folder).Error; err != nil {
		return "", err
	}
	return folder.ID.String(), nil
}

// GetFolder implements domain.FolderRepository.
// Subfolders preloaded exclude the special trash folder.
func (f *FolderRepository) GetFolder(ctx context.Context, folder_id uuid.UUID) (*model.FolderModel, error) {
	var folder model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Preload("Folders", "name != ?", constant.TRASH_FOLDER_NAME).
		Preload("Files").
		Where("id = ?", folder_id).First(&folder).Error
	return &folder, err
}

// GetRootFolder returns the user's root folder, creating it if missing.
func (f *FolderRepository) GetRootFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	root := model.FolderModel{}

	err := f.db.DB.WithContext(ctx).Preload("Folders").Preload("Files").
		Where("name = ? AND user_id = ?", constant.ROOT_FOLDER_NAME, user_id).First(&root).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		path := "/" + user_id + "/" + constant.ROOT_FOLDER_NAME

		if err := f.db.DB.WithContext(ctx).Create(&model.FolderModel{
			UserID:      user_id,
			Name:        constant.ROOT_FOLDER_NAME,
			Description: "Root folder",
			Path:        path,
		}).Error; err != nil {
			return nil, err
		}

		return f.GetRootFolder(ctx, user_id)
	}

	return &root, nil
}

// GetTrashFolder returns the user's trash folder, creating it if missing.
// The trash folder is a top-level reserved folder where deleted items are moved.
func (f *FolderRepository) GetTrashFolder(ctx context.Context, user_id string) (*model.FolderModel, error) {
	trash := model.FolderModel{}

	err := f.db.DB.WithContext(ctx).
		Where("name = ? AND user_id = ?", constant.TRASH_FOLDER_NAME, user_id).First(&trash).Error
	if err == nil {
		return &trash, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	path := "/" + user_id + "/" + constant.TRASH_FOLDER_NAME
	if err := f.db.DB.WithContext(ctx).Create(&model.FolderModel{
		UserID:      user_id,
		Name:        constant.TRASH_FOLDER_NAME,
		Description: "Trash",
		Path:        path,
	}).Error; err != nil {
		return nil, err
	}

	return f.GetTrashFolder(ctx, user_id)
}

// GetFolders implements domain.FolderRepository.
// Excludes the special trash folder from results.
func (f *FolderRepository) GetFolders(ctx context.Context, parent_id uuid.UUID) ([]model.FolderModel, error) {
	var folders []model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ? AND name != ?", parent_id, constant.TRASH_FOLDER_NAME).
		Find(&folders).Error
	return folders, err
}

// GetFoldersPaginated implements domain.FolderRepository.
// Excludes the special trash folder from results.
func (f *FolderRepository) GetFoldersPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]model.FolderModel, error) {
	page.Validate()
	var folders []model.FolderModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ? AND name != ?", parent_id, constant.TRASH_FOLDER_NAME).
		Offset(page.Offset()).
		Limit(page.PageSize).
		Order("created_at DESC").
		Find(&folders).Error
	return folders, err
}

// MoveFolder implements domain.FolderRepository.
func (f *FolderRepository) MoveFolder(ctx context.Context, folder_id uuid.UUID, new_parent_id uuid.UUID) error {
	// Reject self-move outright
	if folder_id == new_parent_id {
		return fmt.Errorf("invalid move: cannot move a folder into itself")
	}

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

	// Reject move into self by path equality (defense in depth)
	if newParentFolder.Path == folder.Path {
		return fmt.Errorf("invalid move: cannot move a folder into itself")
	}

	// Prevent moving into its own subfolder (descendant)
	if strings.HasPrefix(newParentFolder.Path, folder.Path+"/") {
		return fmt.Errorf("invalid move: cannot move a folder into its own subfolder")
	}

	oldPath := folder.Path
	newPath := strings.TrimSuffix(newParentFolder.Path, "/") + "/" + folder.Name

	if folder.Path == newPath && folder.ParentID == &newParentFolder.ID {
		return nil
	}

	return f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"path":      newPath,
			"parent_id": newParentFolder.ID,
		}).Error; err != nil {
			return err
		}

		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"
		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// RenameFolder implements domain.FolderRepository.
func (f *FolderRepository) RenameFolder(ctx context.Context, user_id string, folder_id uuid.UUID, new_name string) error {
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("folder not found")
		}
		return err
	}

	oldPath := folder.Path
	newPath := strings.TrimSuffix(folder.Path, "/"+folder.Name) + "/" + new_name

	return f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"name": new_name,
			"path": newPath,
		}).Error; err != nil {
			return err
		}

		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"
		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteFolder moves the folder to the user's trash folder. If the folder is
// already inside the trash, it is permanently deleted instead.
//
// Returns the parent_id, the (oldPath, newPath) of the folder itself, and a
// list of (oldPath, newPath) pairs for every file under the folder so the
// caller can physically move them on the storage backend. If newPath is empty,
// the folder was permanently deleted and the caller should delete the prefix
// from the backend instead.
func (f *FolderRepository) DeleteFolder(ctx context.Context, folder_id uuid.UUID) (parent_id, oldPath, newPath string, fileMoves []model.PathMove, err error) {
	folder, err := f.GetFolder(ctx, folder_id)
	if err != nil {
		return "", "", "", nil, err
	}

	if folder.ParentID != nil {
		parent_id = folder.ParentID.String()
	}

	// Get the user's trash folder
	trash, err := f.GetTrashFolder(ctx, folder.UserID)
	if err != nil {
		return parent_id, "", "", nil, err
	}

	// If the folder IS the trash, refuse
	if folder.ID == trash.ID {
		return parent_id, "", "", nil, fmt.Errorf("cannot delete the trash folder itself")
	}

	oldPath = folder.Path

	// If the folder is already inside the trash, permanently delete it
	if strings.HasPrefix(folder.Path, trash.Path+"/") || (folder.ParentID != nil && *folder.ParentID == trash.ID) {
		_, scrubErr := f.ScrubFolder(ctx, folder.UserID, folder_id)
		return parent_id, oldPath, "", nil, scrubErr
	}

	// Move folder into trash, with a unique name to avoid collisions
	newName := uniqueTrashName(ctx, f.db.DB, trash.ID, folder.Name)
	newPath = trash.Path + "/" + newName

	txErr := f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Collect all descendant file paths BEFORE the rewrite so we can return
		// the (oldPath, newPath) pairs to the caller
		oldPrefix := oldPath + "/"
		newPrefix := newPath + "/"

		var files []model.FileModel
		if err := tx.Where("path LIKE ?", oldPrefix+"%").Find(&files).Error; err != nil {
			return err
		}
		for _, file := range files {
			fileMoves = append(fileMoves, model.PathMove{
				Old: file.Path,
				New: newPrefix + strings.TrimPrefix(file.Path, oldPrefix),
			})
		}

		if err := tx.Model(&folder).Updates(map[string]interface{}{
			"name":      newName,
			"path":      newPath,
			"parent_id": trash.ID,
		}).Error; err != nil {
			return err
		}

		prefixLen := len(oldPrefix)

		if err := tx.Model(&model.FolderModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.FileModel{}).
			Where("path LIKE ?", oldPrefix+"%").
			Update("path", gorm.Expr("? || SUBSTR(path, ?)", newPrefix, prefixLen+1)).Error; err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return parent_id, "", "", nil, txErr
	}

	return parent_id, oldPath, newPath, fileMoves, nil
}


// ScrubFolder permanently deletes a folder and all its contents.
func (f *FolderRepository) ScrubFolder(ctx context.Context, user_id string, folder_id uuid.UUID) (parent_id string, err error) {
	var folder model.FolderModel
	if err := f.db.DB.WithContext(ctx).Where("id = ? AND user_id = ?", folder_id, user_id).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("folder not found")
		}
		return "", err
	}

	if err := f.db.DB.WithContext(ctx).Delete(&folder).Error; err != nil {
		return "", err
	}

	if folder.ParentID != nil {
		return folder.ParentID.String(), nil
	}
	return "", nil
}

// uniqueTrashName returns a name that doesn't collide with existing items in the trash folder.
// If "photo.png" exists, returns "photo (2).png", "photo (3).png", etc.
func uniqueTrashName(ctx context.Context, db *gorm.DB, trashID uuid.UUID, name string) string {
	var count int64
	db.WithContext(ctx).Model(&model.FolderModel{}).
		Where("parent_id = ? AND name = ?", trashID, name).Count(&count)
	if count == 0 {
		return name
	}

	// Try suffixes until we find one that's free
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)", name, i)
		db.WithContext(ctx).Model(&model.FolderModel{}).
			Where("parent_id = ? AND name = ?", trashID, candidate).Count(&count)
		if count == 0 {
			return candidate
		}
	}
	// Fallback: use a random uuid suffix
	return name + "-" + uuid.New().String()[:8]
}
