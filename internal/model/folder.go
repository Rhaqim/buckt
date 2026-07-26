package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`                                     // Unique identifier for the file
	UserID      string     `gorm:"not null;uniqueIndex:idx_folder_user_parent_name" json:"user_id"`    // ID of the user who owns the bucket
	ParentID    *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_folder_user_parent_name" json:"parent_id"` // Foreign key to FolderModel
	Name        string     `gorm:"not null;uniqueIndex:idx_folder_user_parent_name" json:"name"`       // Folder name
	Description string     `gorm:"type:text" json:"description"`                                       // Optional description of the bucket
	Path        string     `gorm:"not null;unique" json:"path"`                                        // File path
	// OriginParentID records the parent folder this folder lived in before it was
	// moved to trash, so restore returns it to its original location instead of
	// root. Nil when not in trash (or for items trashed before this was tracked —
	// those restore to root). Plain nullable column, no FK.
	OriginParentID *uuid.UUID    `gorm:"type:uuid" json:"origin_parent_id,omitempty"`
	Folders        []FolderModel `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"folders"` // Establish one-to-many relationship with FolderModel
	Files          []FileModel   `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"files"`   // Establish one-to-many relationship with FileModel
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`

	// Deprecated: soft-delete was replaced by the __trash__ folder. Retained only
	// for source compatibility with pre-trash releases — fully ignored by the
	// database (gorm:"-", so no deleted_at column and no soft-delete filtering)
	// and never populated (always Valid == false). Use Client.GetTrashFolder to
	// enumerate trashed items. See FileModel.DeletedAt for why this is
	// sql.NullTime rather than gorm.DeletedAt.
	DeletedAt sql.NullTime `gorm:"-" json:"deleted_at"`
}

// BeforeCreate hook for FolderModel to add a prefixed UUID
func (folder *FolderModel) BeforeCreate(tx *gorm.DB) (err error) {
	folder.ID = uuid.New()
	return
}
