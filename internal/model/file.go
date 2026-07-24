package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`                                       // File ID
	UserID      string    `gorm:"not null;index;default:''" json:"user_id"`                             // Owner of the file
	Name        string    `gorm:"not null;uniqueIndex:idx_file_parent_name" json:"name"`                // File name
	Path        string    `gorm:"not null;unique" json:"path"`                                          // File path
	ContentType string    `gorm:"not null" json:"content_type"`                                         // MIME type (e.g., image/png, application/pdf)
	Size        int64     `gorm:"not null" json:"size"`                                                 // File size in bytes
	ParentID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_file_parent_name" json:"parent_id"` // Foreign key to FolderModel
	Hash        string    `gorm:"not null;index" json:"hash"`                                           // Hash of the file for integrity checks and uniqueness
	Data        []byte    `gorm:"-" json:"data"`                                                        // File data
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Deprecated: soft-delete was replaced by the __trash__ folder. This field
	// is retained only for source compatibility with pre-trash releases — it is
	// fully ignored by the database (gorm:"-:all", so no deleted_at column and no
	// soft-delete filtering) and is never populated (always reports Valid ==
	// false). Enumerate trashed items via Client.GetTrashFolder instead.
	DeletedAt gorm.DeletedAt `gorm:"-:all" json:"deleted_at"`
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = uuid.New()
	return
}
