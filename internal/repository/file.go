package repository

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileRepository struct {
	*database.DB
	*logger.BucktLogger
}

func NewFileRepository(db *database.DB, logger *logger.BucktLogger) domain.FileRepository {
	return &FileRepository{db, logger}
}

// Create implements domain.FileRepository.
// Subtle: this method shadows the method (*DB).Create of FileRepository.DB.
func (f *FileRepository) Create(file *model.FileModel) error {
	return f.DB.Create(file).Error
}

// RestoreFileByPath implements domain.FileRepository.
// if it already exists, overwrite it and set the deleted_at to nil
func (f *FileRepository) RestoreFile(parent_id uuid.UUID, name string) (*model.FileModel, error) {
	var file model.FileModel
	err := f.DB.Unscoped().Model(&model.FileModel{}).Where("parent_id = ? AND name = ?", parent_id, name).Update("deleted_at", nil).Scan(&file).Error

	return &file, err
}

// GetFile implements domain.FileRepository.
func (f *FileRepository) GetFile(id uuid.UUID) (*model.FileModel, error) {
	var file model.FileModel
	err := f.DB.First(&file, id).Error
	return &file, err
}

// GetFiles implements domain.FileRepository.
func (f *FileRepository) GetFiles(parent_id uuid.UUID) ([]*model.FileModel, error) {
	var files []*model.FileModel
	err := f.DB.Where("parent_id = ?", parent_id).Find(&files).Error
	return files, err
}

// MoveFile implements domain.FileRepository.
func (f *FileRepository) MoveFile(file_id uuid.UUID, new_parent_id uuid.UUID) (string, string, error) { // TODO: MOdify function to accept file_id, new_parent_id, and new_name
	var newParentFolder model.FolderModel

	if err := f.DB.Where("id = ?", new_parent_id).First(&newParentFolder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("parent folder not found")
		}
		return "", "", err
	}

	var file model.FileModel
	if err := f.DB.First(&file, file_id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("file not found")
		}
		return "", "", err
	}

	oldPath := file.Path

	file.ParentID = new_parent_id
	file.Path = newParentFolder.Path + "/" + file.Name

	if err := f.DB.Save(&file).Error; err != nil {
		return "", "", err
	}

	return oldPath, file.Path, nil
}

// RenameFile implements domain.FileRepository.
func (f *FileRepository) RenameFile(file_id uuid.UUID, new_name string) error {
	var file model.FileModel
	if err := f.DB.First(&file, file_id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("file not found")
		}
		return err
	}

	file.Name = new_name

	return f.DB.Save(&file).Error
}

// Update implements domain.FileRepository.
// Subtle: this method shadows the method (*DB).Update of FileRepository.DB.
func (f *FileRepository) Update(file *model.FileModel) error {
	return f.DB.Save(file).Error
}

// DeleteFile implements domain.FileRepository.
func (f *FileRepository) DeleteFile(id uuid.UUID) error {
	return f.DB.Delete(&model.FileModel{}, id).Error
}

// ScrubFile implements domain.FileRepository.
func (f *FileRepository) ScrubFile(id uuid.UUID) error {
	return f.DB.Unscoped().Delete(&model.FileModel{}, id).Error
}
