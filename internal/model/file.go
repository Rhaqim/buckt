package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileModel struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"` // Unique identifier for the file
	Name        string    `gorm:"not null"`             // File name
	Path        string    `gorm:"not null;unique"`      // File path
	ContentType string    `gorm:"not null"`             // MIME type (e.g., image/png, application/pdf)
	Size        int64     `gorm:"not null"`             // File size in bytes
	ParentID    uuid.UUID `gorm:"type:uuid;not null"`   // Foreign key to FolderModel
	Hash        string    `gorm:"not null;unique"`      // Hash of the file for integrity checks and uniqueness
	Data        []byte    `gorm:"-"`                    // File data
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = uuid.New()
	return
}
