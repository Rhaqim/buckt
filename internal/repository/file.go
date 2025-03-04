package repository

import (
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
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
func (f *FileRepository) RestoreFile(hash string) (*model.FileModel, error) {
	var file model.FileModel
	err := f.DB.Unscoped().Model(&model.FileModel{}).Where("hash = ?", hash).Update("deleted_at", nil).Scan(&file).Error

	return &file, err
}

// DeleteFile implements domain.FileRepository.
func (f *FileRepository) DeleteFile(id uuid.UUID) error {
	return f.DB.Delete(&model.FileModel{}, id).Error
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

// Update implements domain.FileRepository.
// Subtle: this method shadows the method (*DB).Update of FileRepository.DB.
func (f *FileRepository) Update(file *model.FileModel) error {
	return f.DB.Save(file).Error
}
