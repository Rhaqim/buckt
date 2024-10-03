package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileModel struct {
	ID          string     `gorm:"type:uuid;primaryKey"` // Unique identifier for the file
	Name        string     `gorm:"not null"`             // File name
	Path        string     `gorm:"not null"`             // Full path or URL to the file
	ContentType string     `gorm:"not null"`             // MIME type (e.g., image/png, application/pdf)
	Size        int64      `gorm:"not null"`             // File size in bytes
	BucketID    string     `gorm:"type:uuid;not null"`   // Foreign key to BucketModel
	Hash        string     `gorm:"not null;unique"`      // Hash of the file for integrity checks and uniqueness
	Tags        []TagModel `gorm:"many2many:file_tags;"` // Establish many-to-many relationship with TagModel
	gorm.Model
}

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) domain.Repository[FileModel] {
	return &FileRepository{db}
}

func (r *FileRepository) Create(file *FileModel) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) FindAll() ([]FileModel, error) {
	var files []FileModel
	err := r.db.Find(&files).Error
	return files, err
}

func (r *FileRepository) FindByID(id string) (FileModel, error) {
	var file FileModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *FileRepository) Delete(id string) error {
	return r.db.Delete(&FileModel{}, id).Error
}

func (r *FileRepository) GetBy(key string, value string) (FileModel, error) {
	var file FileModel
	err := r.db.Where(key+" = ?", value).First(&file).Error
	return file, err
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = "file-" + uuid.New().String()
	return
}
