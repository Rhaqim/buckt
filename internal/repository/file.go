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

type FileRepository struct {
	db *database.DB
}

func NewFileRepository(db *database.DB) domain.FileRepository {
	return &FileRepository{db: db}
}

// Create implements domain.FileRepository.
func (f *FileRepository) Create(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Create(file).Error
}

// GetFile implements domain.FileRepository.
func (f *FileRepository) GetFile(ctx context.Context, id uuid.UUID) (*model.FileModel, error) {
	var file model.FileModel
	err := f.db.DB.WithContext(ctx).First(&file, id).Error
	return &file, err
}

// GetFiles implements domain.FileRepository.
func (f *FileRepository) GetFiles(ctx context.Context, parent_id uuid.UUID) ([]*model.FileModel, error) {
	var files []*model.FileModel
	err := f.db.DB.WithContext(ctx).Where("parent_id = ?", parent_id).Find(&files).Error
	return files, err
}

// GetFilesPaginated implements domain.FileRepository.
func (f *FileRepository) GetFilesPaginated(ctx context.Context, parent_id uuid.UUID, page model.Pagination) ([]*model.FileModel, error) {
	page.Validate()
	var files []*model.FileModel
	err := f.db.DB.WithContext(ctx).
		Where("parent_id = ?", parent_id).
		Offset(page.Offset()).
		Limit(page.PageSize).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

// MoveFile implements domain.FileRepository.
func (f *FileRepository) MoveFile(ctx context.Context, file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) {
	var newParentFolder model.FolderModel

	if err := f.db.DB.WithContext(ctx).Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", fmt.Errorf("parent folder not found")
		}
		return "", "", err
	}

	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, file_id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", fmt.Errorf("file not found")
		}
		return "", "", err
	}

	oldPath := file.Path

	file.ParentID = new_parent_id
	file.Path = newParentFolder.Path + "/" + file.Name

	if err := f.db.DB.WithContext(ctx).Save(&file).Error; err != nil {
		return "", "", err
	}

	return oldPath, file.Path, nil
}

// RenameFile implements domain.FileRepository.
func (f *FileRepository) RenameFile(ctx context.Context, file_id uuid.UUID, new_name string) error {
	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, file_id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("file not found")
		}
		return err
	}

	file.Name = new_name

	return f.db.DB.WithContext(ctx).Save(&file).Error
}

// Update implements domain.FileRepository.
func (f *FileRepository) Update(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Save(file).Error
}

// DeleteFile moves the file to the user's trash folder. If the file is already
// in trash, it is permanently deleted instead.
//
// Returns the file's old and new paths so the caller can move the underlying
// data on the storage backend. If newPath is empty, the file was permanently
// deleted (already in trash). If newPath equals oldPath, the file was moved
// in flat-namespace mode and no backend op is needed.
//
// The beforeCommit callback runs INSIDE the DB transaction after the path
// rewrite but before commit. If the callback returns an error, the entire
// transaction rolls back, leaving the DB in its original state. This makes
// the DB and backend stay consistent even if the backend operation fails.
func (f *FileRepository) DeleteFile(
	ctx context.Context,
	id uuid.UUID,
	beforeCommit func(oldPath, newPath string) error,
) (oldPath, newPath string, err error) {
	err = f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var file model.FileModel
		if err := tx.First(&file, id).Error; err != nil {
			return err
		}

		oldPath = file.Path

		// Defense in depth: if user_id is empty (legacy data not yet backfilled),
		// derive it from the parent folder so we don't move files into a phantom
		// "empty user" trash bin.
		if file.UserID == "" {
			var parent model.FolderModel
			if err := tx.Select("user_id").First(&parent, file.ParentID).Error; err == nil && parent.UserID != "" {
				file.UserID = parent.UserID
				_ = tx.Model(&file).Update("user_id", parent.UserID).Error
			}
		}

		// If we still don't have a user_id, refuse to delete.
		if file.UserID == "" {
			return fmt.Errorf("cannot delete file %s: unable to resolve owner user_id (legacy row with missing user_id and no parent folder user_id)", id)
		}

		// Lookup or create the user's trash folder
		trash, err := getOrCreateTrashFolder(ctx, tx, file.UserID)
		if err != nil {
			return err
		}

		// If the file is already in trash, permanently delete it
		if file.ParentID == trash.ID {
			if err := tx.Delete(&model.FileModel{}, id).Error; err != nil {
				return err
			}
			// Run backend cleanup BEFORE commit — failure rolls back the row deletion
			if beforeCommit != nil {
				if err := beforeCommit(oldPath, ""); err != nil {
					return err
				}
			}
			return nil
		}

		// Move file into trash with a unique name
		newName, err := uniqueFileTrashName(ctx, tx, trash.ID, file.Name)
		if err != nil {
			return err
		}

		// Detect flat-namespace mode by checking for any directory separator
		// (either / or \, since filepath.Join on Windows can produce backslashes).
		isFlatPath := !strings.ContainsAny(file.Path, "/\\")

		updates := map[string]any{
			"name":      newName,
			"parent_id": trash.ID,
		}
		if !isFlatPath {
			newPath = trash.Path + "/" + newName
			updates["path"] = newPath
		} else {
			// Path stays the same in flat mode; signal "no backend move needed"
			// by returning the same value for old and new.
			newPath = file.Path
		}

		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			return err
		}

		// Run backend operation BEFORE commit. If it fails, the transaction
		// rolls back and the path rewrite is undone.
		if beforeCommit != nil {
			if err := beforeCommit(oldPath, newPath); err != nil {
				return err
			}
		}
		return nil
	})
	return
}

// ScrubFile permanently deletes a file regardless of its location.
func (f *FileRepository) ScrubFile(ctx context.Context, id uuid.UUID) error {
	return f.db.DB.WithContext(ctx).Delete(&model.FileModel{}, id).Error
}

// getOrCreateTrashFolder returns the user's trash folder, creating it if missing.
// This duplicates the FolderRepository logic to avoid a cross-repo dependency.
func getOrCreateTrashFolder(ctx context.Context, db *gorm.DB, user_id string) (*model.FolderModel, error) {
	const trashName = constant.TRASH_FOLDER_NAME
	var trash model.FolderModel

	err := db.WithContext(ctx).
		Where("name = ? AND user_id = ?", trashName, user_id).First(&trash).Error
	if err == nil {
		return &trash, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	path := "/" + user_id + "/" + trashName
	trash = model.FolderModel{
		UserID:      user_id,
		Name:        trashName,
		Description: "Trash",
		Path:        path,
	}
	if err := db.WithContext(ctx).Create(&trash).Error; err != nil {
		return nil, err
	}
	return &trash, nil
}

// uniqueFileTrashName returns a file name that doesn't collide with existing
// files in the trash folder. If "photo.png" already exists, it returns
// "photo (2).png", "photo (3).png", etc., preserving the extension.
func uniqueFileTrashName(ctx context.Context, db *gorm.DB, trashID uuid.UUID, name string) (string, error) {
	count, err := countFileName(ctx, db, trashID, name)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return name, nil
	}

	// Split into base and extension to insert the suffix sensibly
	ext := ""
	base := name
	if idx := strings.LastIndex(name, "."); idx > 0 {
		ext = name[idx:]
		base = name[:idx]
	}

	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		count, err := countFileName(ctx, db, trashID, candidate)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return base + "-" + uuid.New().String()[:8] + ext, nil
}

func countFileName(ctx context.Context, db *gorm.DB, parentID uuid.UUID, name string) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.FileModel{}).
		Where("parent_id = ? AND name = ?", parentID, name).Count(&count).Error
	return count, err
}
