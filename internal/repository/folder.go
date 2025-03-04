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
	*logger.BucktLogger
}

func NewFolderRepository(db *database.DB, logger *logger.BucktLogger) domain.FolderRepository {
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

// GetRootFolder implements domain.FolderRepository.
// look for a folder called root and return it, if root doesn't exist, create it
func (f *FolderRepository) GetRootFolder(user_id string) (*model.FolderModel, error) {
	root := model.FolderModel{}

	root_folder := "root_folder"

	err := f.DB.Preload("Folders").Preload("Files").Where("name = ? AND user_id = ?", root_folder, user_id).First(&root).Error
	if err != nil {
		if err.Error() != "record not found" {
			return nil, err
		}

		path := "/" + user_id + "/" + root_folder

		if err := f.DB.Create(&model.FolderModel{
			UserID:      user_id,
			Name:        root_folder,
			Description: "Root folder",
			Path:        path,
		}).Error; err != nil {
			return nil, err
		}

		return f.GetRootFolder(user_id)
	}

	return &root, nil
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
