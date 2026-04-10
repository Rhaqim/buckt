package repository

import (
	"context"
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
func (f *FileRepository) Update(ctx context.Context, file *model.FileModel) error {
	return f.db.DB.WithContext(ctx).Save(file).Error
}

// DeleteFile moves the file to the user's trash folder. If the file is already
// in trash, it is permanently deleted instead.
//
// Returns the file's old and new paths so the caller can move the underlying
// data on the storage backend. If newPath is empty, the file was permanently
// deleted (already in trash) and no backend move is needed — the caller should
// call backend.Delete instead.
func (f *FileRepository) DeleteFile(ctx context.Context, id uuid.UUID) (oldPath, newPath string, err error) {
	var file model.FileModel
	if err := f.db.DB.WithContext(ctx).First(&file, id).Error; err != nil {
		return "", "", err
	}

	oldPath = file.Path

	// Lookup or create the user's trash folder
	trash, err := getOrCreateTrashFolder(ctx, f.db.DB, file.UserID)
	if err != nil {
		return "", "", err
	}

	// If the file is already in trash, permanently delete it
	if file.ParentID == trash.ID {
		if err := f.db.DB.WithContext(ctx).Delete(&model.FileModel{}, id).Error; err != nil {
			return "", "", err
		}
		return oldPath, "", nil
	}

	// Move file into trash with a unique name
	newName := uniqueFileTrashName(ctx, f.db.DB, trash.ID, file.Name)
	newPath = trash.Path + "/" + newName

	if err := f.db.DB.WithContext(ctx).Model(&file).Updates(map[string]interface{}{
		"name":      newName,
		"path":      newPath,
		"parent_id": trash.ID,
	}).Error; err != nil {
		return "", "", err
	}
	return oldPath, newPath, nil
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
	if err != gorm.ErrRecordNotFound && err.Error() != "record not found" {
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

// uniqueFileTrashName returns a name that doesn't collide with existing files in the trash folder.
func uniqueFileTrashName(ctx context.Context, db *gorm.DB, trashID uuid.UUID, name string) string {
	var count int64
	db.WithContext(ctx).Model(&model.FileModel{}).
		Where("parent_id = ? AND name = ?", trashID, name).Count(&count)
	if count == 0 {
		return name
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
		db.WithContext(ctx).Model(&model.FileModel{}).
			Where("parent_id = ? AND name = ?", trashID, candidate).Count(&count)
		if count == 0 {
			return candidate
		}
	}
	return base + "-" + uuid.New().String()[:8] + ext
}
