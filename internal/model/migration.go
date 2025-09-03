package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MigrationBackend string

const (
	MigrationBackendLocal MigrationBackend = "local"
	MigrationBackendS3    MigrationBackend = "s3"
)

type MigrationStatus string

const (
	MigrationStatusQueued    MigrationStatus = "queued"
	MigrationStatusUploading MigrationStatus = "uploading"
	MigrationStatusVerifying MigrationStatus = "verifying"
	MigrationStatusCommitted MigrationStatus = "committed"
	MigrationStatusFailed    MigrationStatus = "failed"
)

type MigrationModel struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	FileID         uuid.UUID        `gorm:"type:uuid" json:"file_id"`
	Backend        MigrationBackend `gorm:"backend" json:"backend"`
	ObjectKey      string           `gorm:"object_key" json:"object_key"`
	Size           int64            `gorm:"size" json:"size"`
	ChecksumSHA256 string           `gorm:"checksum_sha256" json:"checksum_sha256"`
	Status         MigrationStatus  `gorm:"status" json:"status"`
	Version        int              `gorm:"version" json:"version"`
	CreatedAt      time.Time        `gorm:"created_at" json:"created_at"`
	UpdatedAt      time.Time        `gorm:"updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `gorm:"index" json:"deleted_at"`
}

// BeforeCreate hook for MigrationModel to add a prefixed UUID
func (migration *MigrationModel) BeforeCreate(tx *gorm.DB) (err error) {
	migration.ID = uuid.New()
	return
}
