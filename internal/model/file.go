package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`                                  // File ID
	Name        string         `gorm:"not null;uniqueIndex:idx_parent_name" json:"name"`                // File name
	Path        string         `gorm:"not null;unique" json:"path"`                                     // File path
	ContentType string         `gorm:"not null" json:"content_type"`                                    // MIME type (e.g., image/png, application/pdf)
	Size        int64          `gorm:"not null" json:"size"`                                            // File size in bytes
	ParentID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_parent_name" json:"parent_id"` // Foreign key to FolderModel
	Hash        string         `gorm:"not null;index" json:"hash"`                                      // Hash of the file for integrity checks and uniqueness
	Data        []byte         `gorm:"-" json:"data"`                                                   // File data
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = uuid.New()
	return
}
