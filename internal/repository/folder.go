package repository

import (
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type FolderRepository struct {
	*database.DB
	*logger.Logger
}

func NewFolderRepository(db *database.DB, logger *logger.Logger) domain.FolderRepository {
	return &FolderRepository{db, logger}
}

// Create implements domain.FolderRepository.
// Subtle: this method shadows the method (*DB).Create of FolderRepository.DB.
func (f *FolderRepository) Create(folder *model.FolderModel) error {
	return f.DB.Create(folder).Error
}

// GetFolder implements domain.FolderRepository.
func (f *FolderRepository) GetFolder(folder_id uuid.UUID) (*model.FolderModel, error) {
	var folder model.FolderModel
	err := f.DB.Preload("Folders").Preload("Files").Where("id = ?", folder_id).First(&folder).Error
	return &folder, err
}

// GetFolders implements domain.FolderRepository.
func (f *FolderRepository) GetFolders(bucket_id uuid.UUID) ([]model.FolderModel, error) {
	var folders []model.FolderModel
	err := f.DB.Preload("Folders").Preload("Files").Where("bucket_id = ?", bucket_id).Find(&folders).Error
	return folders, err
}

// MoveFolder implements domain.FolderRepository.
func (f *FolderRepository) MoveFolder(folder_id uuid.UUID, new_parent_id uuid.UUID) error {
	return f.DB.Model(&model.FolderModel{}).Where("id = ?", folder_id).Update("parent_id", new_parent_id).Error
}

// RenameFolder implements domain.FolderRepository.
func (f *FolderRepository) RenameFolder(folder_id uuid.UUID, new_name string) error {
	return f.DB.Model(&model.FolderModel{}).Where("id = ?", folder_id).Update("name", new_name).Error
}
