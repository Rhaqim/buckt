package repository

import (
	"context"
	"fmt"

	"github.com/Rhaqim/buckt/internal/database"
	"strings"
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
// Subtle: this method shadows the method (*DB).Create of FileRepository.DB.WithContext(ctx).
func (f *FileRepository) Create(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Create(file).Error
}

// RestoreFile finds a soft-deleted file by its original name and restores it.
// Handles both mangled names (new deletes) and unmangled names (legacy deletes).
func (f *FileRepository) RestoreFile(ctx context.Context, parent_id uuid.UUID, name string) (*model.FileModel, error) {
	var file model.FileModel

	// Try mangled name first (new-style soft-delete)
	err := f.db.DB.WithContext(ctx).Unscoped().
		Where("parent_id = ? AND name LIKE ? AND deleted_at IS NOT NULL", parent_id, name+".__deleted_%").
		First(&file).Error

	if err != nil {
		// Fall back to exact name match (legacy soft-delete without mangling)
		err = f.db.DB.WithContext(ctx).Unscoped().
			Where("parent_id = ? AND name = ? AND deleted_at IS NOT NULL", parent_id, name).
			First(&file).Error
		if err != nil {
			return nil, err
		}
	}

	// Restore: ensure name and path are the clean originals
	suffix := ".__deleted_" + file.ID.String()
	originalName := strings.TrimSuffix(file.Name, suffix)
	originalPath := strings.TrimSuffix(file.Path, suffix)

	err = f.db.DB.WithContext(ctx).Unscoped().Model(&file).Updates(map[string]interface{}{
		"deleted_at": nil,
		"name":       originalName,
		"path":       originalPath,
	}).Error
	if err != nil {
		return nil, err
	}

	file.Name = originalName
	file.Path = originalPath
	return &file, nil
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
func (f *FileRepository) MoveFile(ctx context.Context, file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) { // TODO: MOdify function to accept file_id, new_parent_id, and new_name
	var newParentFolder model.FolderModel

	if err := f.db.DB.WithContext(ctx).Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("parent folder not found")
		}
		return "", "", err
	}

	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, file_id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("file not found")
		}
		return err
	}

	file.Name = new_name

	return f.db.DB.WithContext(ctx).Save(&file).Error
}

// Update implements domain.FileRepository.
// Subtle: this method shadows the method (*DB).Update of FileRepository.DB.WithContext(ctx).
func (f *FileRepository) Update(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Save(file).Error
}

// DeleteFile implements domain.FileRepository.
func (f *FileRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	return f.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var file model.FileModel
		if err := tx.First(&file, id).Error; err != nil {
			return err
		}

		// Mangle path and name to free unique constraints for future files with the same name
		suffix := ".__deleted_" + file.ID.String()
		if err := tx.Model(&file).Updates(map[string]any{
			"path": file.Path + suffix,
			"name": file.Name + suffix,
		}).Error; err != nil {
			return err
		}

		return tx.Delete(&file).Error
	})
}

// ScrubFile implements domain.FileRepository.
func (f *FileRepository) ScrubFile(ctx context.Context, id uuid.UUID) error {
	return f.db.DB.WithContext(ctx).Unscoped().Delete(&model.FileModel{}, id).Error
}
