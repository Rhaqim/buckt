package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	gorm.Model
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey"`                            // Unique identifier for the file
	UserID      string        `gorm:"not null"`                                        // ID of the user who owns the bucket
	ParentID    *uuid.UUID    `gorm:"type:uuid"`                                       // Foreign key to FolderModel
	Name        string        `gorm:"not null;unique"`                                 // Folder name
	Description string        `gorm:"type:text"`                                       // Optional description of the bucket
	Path        string        `gorm:"not null;unique"`                                 // File path
	Folders     []FolderModel `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"` // Establish one-to-many relationship with FolderModel
	Files       []FileModel   `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"` // Establish one-to-many relationship with FileModel
}

// BeforeCreate hook for FolderModel to add a prefixed UUID
func (folder *FolderModel) BeforeCreate(tx *gorm.DB) (err error) {
	folder.ID = uuid.New()
	return
}
