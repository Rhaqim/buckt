package model

import (
	"database/sql"
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

	// Deprecated: soft-delete was replaced by the __trash__ folder. Retained only
	// for source compatibility with pre-trash releases — fully ignored by the
	// database (gorm:"-", so no deleted_at column and no soft-delete filtering)
	// and never populated (always reports Valid == false). Enumerate trashed
	// items via Client.GetTrashFolder instead.
	//
	// Note: the type is sql.NullTime, not gorm.DeletedAt. gorm.DeletedAt is
	// defined as `type DeletedAt sql.NullTime`, so the .Time/.Valid field access
	// and JSON shape are identical, but a gorm.DeletedAt-typed field — even when
	// tagged as ignored — makes gorm emit a broken "deleted_at IS NULL" clause on
	// every query. sql.NullTime avoids that entirely.
	DeletedAt sql.NullTime `gorm:"-" json:"deleted_at"`
}

// BeforeCreate hook for FileModel to add a prefixed UUID
func (file *FileModel) BeforeCreate(tx *gorm.DB) (err error) {
	file.ID = uuid.New()
	return
}
