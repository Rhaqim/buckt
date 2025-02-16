package service

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type FolderService struct {
	*logger.Logger

	domain.FolderRepository
}

func NewFolderService(log *logger.Logger, folderRepository domain.FolderRepository) domain.FolderService {
	return &FolderService{
		Logger:           log,
		FolderRepository: folderRepository,
	}
}

// CreateFolder implements domain.FolderService.
func (f *FolderService) CreateFolder(user_id, parent_id, folder_name, description string) error {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}
	// Get the parent folder
	parentFolder, err := f.FolderRepository.GetFolder(parentID)
	if err != nil {
		return f.WrapError("failed to get parent folder", err)
	}

	path := parentFolder.Name + "/" + folder_name

	folder := &model.FolderModel{
		UserID:      user_id,
		ParentID:    parentID,
		Name:        folder_name,
		Description: description,
		Path:        path,
	}

	if err := f.FolderRepository.Create(folder); err != nil {
		return f.WrapError("failed to create folder", err)
	}

	return nil
}

// GetFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolder of FolderService.FolderRepository.
func (f *FolderService) GetFolder(folder_id string) (*model.FolderModel, error) {
	id, err := uuid.Parse(folder_id)
	if err != nil {
		return nil, f.WrapError("failed to parse uuid", err)
	}

	folder, err := f.FolderRepository.GetFolder(id)
	if err != nil {
		return nil, f.WrapError("failed to get folder", err)
	}

	return folder, nil
}

// GetFolders implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).GetFolders of FolderService.FolderRepository.
func (f *FolderService) GetFolders(parent_id string) ([]model.FolderModel, error) {
	parentID, err := uuid.Parse(parent_id)
	if err != nil {
		return nil, f.WrapError("failed to parse uuid", err)
	}

	folders, err := f.FolderRepository.GetFolders(parentID)
	if err != nil {
		return nil, f.WrapError("failed to get folders", err)
	}

	return folders, nil
}

// MoveFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).MoveFolder of FolderService.FolderRepository.
func (f *FolderService) MoveFolder(folder_id string, new_parent_id string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}

	newParentID, err := uuid.Parse(new_parent_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}

	if err := f.FolderRepository.MoveFolder(folderID, newParentID); err != nil {
		return f.WrapError("failed to move folder", err)
	}

	return nil
}

// RenameFolder implements domain.FolderService.
// Subtle: this method shadows the method (FolderRepository).RenameFolder of FolderService.FolderRepository.
func (f *FolderService) RenameFolder(folder_id string, new_name string) error {
	folderID, err := uuid.Parse(folder_id)
	if err != nil {
		return f.WrapError("failed to parse uuid", err)
	}

	if err := f.FolderRepository.RenameFolder(folderID, new_name); err != nil {
		return f.WrapError("failed to rename folder", err)
	}

	return nil
}
