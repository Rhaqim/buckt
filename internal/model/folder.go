package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`                                 // Unique identifier for the file
	UserID      string         `gorm:"not null" json:"user_id"`                                        // ID of the user who owns the bucket
	ParentID    *uuid.UUID     `gorm:"type:uuid" json:"parent_id"`                                     // Foreign key to FolderModel
	Name        string         `gorm:"not null;unique" json:"name"`                                    // Folder name
	Description string         `gorm:"type:text" json:"description"`                                   // Optional description of the bucket
	Path        string         `gorm:"not null;unique" json:"path"`                                    // File path
	Folders     []FolderModel  `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"folders"` // Establish one-to-many relationship with FolderModel
	Files       []FileModel    `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"files"`   // Establish one-to-many relationship with FileModel
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// BeforeCreate hook for FolderModel to add a prefixed UUID
func (folder *FolderModel) BeforeCreate(tx *gorm.DB) (err error) {
	folder.ID = uuid.New()
	return
}
