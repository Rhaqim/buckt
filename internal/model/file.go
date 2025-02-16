package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"` // Unique identifier for the file
	Name        string     `gorm:"not null"`             // File name
	Path        string     `gorm:"not null"`             // Full path or URL to the file
	ContentType string     `gorm:"not null"`             // MIME type (e.g., image/png, application/pdf)
	Size        int64      `gorm:"not null"`             // File size in bytes
	ParentID    uuid.UUID  `gorm:"type:uuid;not null"`   // Foreign key to FolderModel
	Hash        string     `gorm:"not null;unique"`      // Hash of the file for integrity checks and uniqueness
	Tags        []TagModel `gorm:"many2many:file_tags;"` // Establish many-to-many relationship with TagModel
	gorm.Model
}

type File struct {
	FileModel
	File []byte
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = uuid.New()
	return
}

// type FileRepository struct {
// 	db *gorm.DB
// }

// func NewFileRepository(db *gorm.DB) domain_old.BucktRepository[FileModel] {
// 	return &FileRepository{db}
// }

// func (r *FileRepository) Create(file *FileModel) error {
// 	return r.db.Create(file).Error
// }

// func (r *FileRepository) Update(file *FileModel) error {
// 	return r.db.Save(file).Error
// }

// func (r *FileRepository) GetAll() ([]FileModel, error) {
// 	var files []FileModel
// 	err := r.db.Find(&files).Error
// 	return files, err
// }

// func (r *FileRepository) GetByID(id uuid.UUID) (FileModel, error) {
// 	var file FileModel
// 	err := r.db.First(&file, id).Error
// 	return file, err
// }

// func (r *FileRepository) Delete(id uuid.UUID) error {
// 	return r.db.Delete(&FileModel{}, id).Error
// }

// func (r *FileRepository) GetBy(key interface{}, value ...interface{}) (FileModel, error) {
// 	var file FileModel
// 	err := r.db.Where(key, value...).First(&file).Error
// 	return file, err
// }

// func (r *FileRepository) GetMany(key interface{}, value ...interface{}) ([]FileModel, error) {
// 	var files []FileModel

// 	err := r.db.Where(key, value).Find(&files).Error
// 	return files, err
// }

// func (r *FileRepository) RawQuery(query string, args ...interface{}) *gorm.DB {
// 	return r.db.Raw(query, args...)
// }
