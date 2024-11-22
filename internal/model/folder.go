package model

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"` // Unique identifier for the file
	Name     string    `gorm:"not null"`             // File name
	BucketID uuid.UUID `gorm:"type:uuid;not null"`   // Foreign key to BucketModel
	ParentID uuid.UUID `gorm:"type:uuid"`            // Foreign key to FolderModel
	gorm.Model
}

type FolderRepository struct {
	db *gorm.DB
}

func NewFolderRepository(db *gorm.DB) *FolderRepository {
	return &FolderRepository{db}
}

func (r *FolderRepository) Create(file *FolderModel) error {
	return r.db.Create(file).Error
}

func (r *FolderRepository) FindAll() ([]FolderModel, error) {
	var files []FolderModel
	err := r.db.Find(&files).Error
	return files, err
}

func (r *FolderRepository) FindByID(id uuid.UUID) (FolderModel, error) {
	var file FolderModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *FolderRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&FolderModel{}, id).Error
}

func (r *FolderRepository) GetBy(key interface{}, value ...interface{}) (FolderModel, error) {
	var file FolderModel
	err := r.db.Where(key, value...).First(&file).Error
	return file, err
}

func (r *FolderRepository) GetMany(key string, value string) ([]FolderModel, error) {
	var files []FolderModel

	err := r.db.Where(key+" = ?", value).Find(&files).Error
	return files, err
}

// BeforeCreate hook for FolderModel to add a prefixed UUID
func (folder *FolderModel) BeforeCreate(tx *gorm.DB) (err error) {
	folder.ID = uuid.New()
	return
}

// GetFullPath retrieves the full path of a folder by recursively traversing up the hierarchy
func (repo *FolderRepository) GetFullPath(folderID uuid.UUID) (string, error) {
	var path string
	query := `
        WITH RECURSIVE folder_hierarchy AS (
			SELECT ID, name, parent_id, name AS path
			FROM folder_models
			WHERE ID = ?
			UNION ALL
			SELECT fm.id, fm.name, fm.parent_id, fm.name || '/' || fh.path AS path
			FROM folder_models fm
			INNER JOIN folder_hierarchy fh ON fh.parent_id = fm.id
		)
		SELECT path 
		FROM folder_hierarchy 
		WHERE parent_id IS NULL;
    `
	if err := repo.db.Raw(query, folderID).Scan(&path).Error; err != nil {
		return "", err
	}
	return path, nil
}

// GetDescendants retrieves all descendants of a folder by traversing down the hierarchy
func (repo *FolderRepository) GetDescendants(folderID uuid.UUID) ([]FolderModel, error) {
	var descendants []FolderModel
	query := `
        WITH RECURSIVE folder_hierarchy AS (
            SELECT id, name, parent_id
            FROM folder_models
            WHERE id = ?
            UNION ALL
            SELECT fm.id, fm.name, fm.parent_id
            FROM folder_models fm
            INNER JOIN folder_hierarchy fh ON fm.parent_id = fh.id
        )
        SELECT * FROM folder_hierarchy;
    `
	if err := repo.db.Raw(query, folderID).Scan(&descendants).Error; err != nil {
		return nil, err
	}
	return descendants, nil
}

func (repo *FolderRepository) GetSubfolders(bucketName, folderPath string) ([]FolderModel, error) {
	// Split the folderPath into individual folder names
	folderNames := strings.Split(folderPath, "/")
	if len(folderNames) < 1 {
		return nil, errors.New("path must include at least one folder")
	}

	// Define the slice to store the result
	var subfolders []FolderModel

	query := `
        WITH RECURSIVE folder_hierarchy AS (
            -- Get the first folder under the specified bucket
            SELECT id, name, parent_id
            FROM folder_models
            WHERE name = ? AND parent_id = (SELECT id FROM bucket_models WHERE name = ?)
            UNION ALL
            SELECT fm.id, fm.name, fm.parent_id
            FROM folder_models fm
            INNER JOIN folder_hierarchy fh ON fm.parent_id = fh.id
        )
        -- Fetch the immediate subfolders of the target folder (the last folder in the path)
        SELECT * 
        FROM folder_models
        WHERE parent_id = (
            SELECT id 
            FROM folder_hierarchy 
            WHERE name = ? 
            -- ORDER BY created_at DESC
            LIMIT 1
        );
    `
	// Run the query with the bucket name, first folder, and last folder name in the path
	if err := repo.db.Raw(query, folderNames[0], bucketName, folderNames[len(folderNames)-1]).Scan(&subfolders).Error; err != nil {
		return nil, err
	}
	return subfolders, nil
}

func (repo *FolderRepository) GetFilesFromPath(bucketName, folderPath string) ([]FileModel, error) {
	// Split the folderPath into individual folder names
	folderNames := strings.Split(folderPath, "/")
	if len(folderNames) < 2 {
		return nil, errors.New("path must include at least one parent and one child folder")
	}

	var files []FileModel

	query := `
        WITH RECURSIVE folder_hierarchy AS (
            SELECT id, name, parent_id
            FROM folder_models
            WHERE name = ? AND parent_id = (SELECT id FROM bucket_models WHERE name = ?)
            UNION ALL
            SELECT fm.id, fm.name, fm.parent_id
            FROM folder_models fm
            INNER JOIN folder_hierarchy fh ON fm.parent_id = fh.id
        )
        SELECT f.* 
        FROM folder_hierarchy fh
        JOIN file_models f ON f.parent_id = fh.id
        WHERE fh.name = ?;
    `

	// Execute query using the first folder name and the bucket name
	if err := repo.db.Raw(query, folderNames[0], bucketName, folderNames[len(folderNames)-1]).Scan(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// MoveFolder moves a folder to a new parent
func (repo *FolderRepository) MoveFolder(folderID, newParentID uuid.UUID) error {
	return repo.db.Model(&FolderModel{}).
		Where("id = ?", folderID).
		Update("parent_id", newParentID).Error
}

// RenameFolder renames a folder
func (repo *FolderRepository) RenameFolder(folderID uuid.UUID, newName string) error {
	return repo.db.Model(&FolderModel{}).
		Where("id = ?", folderID).
		Update("name", newName).Error
}
