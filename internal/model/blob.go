package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlobBackend string

const (
	BlobBackendLocal BlobBackend = "local"
	BlobBackendS3    BlobBackend = "s3"
)

type BlobStatus string

const (
	BlobStatusQueued    BlobStatus = "queued"
	BlobStatusUploading BlobStatus = "uploading"
	BlobStatusVerifying BlobStatus = "verifying"
	BlobStatusCommitted BlobStatus = "committed"
	BlobStatusFailed    BlobStatus = "failed"
)

type BlobModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	FileID         uuid.UUID      `gorm:"type:uuid" json:"file_id"`
	Backend        BlobBackend    `gorm:"backend" json:"backend"`
	ObjectKey      string         `gorm:"object_key" json:"object_key"`
	Size           int64          `gorm:"size" json:"size"`
	ChecksumSHA256 string         `gorm:"checksum_sha256" json:"checksum_sha256"`
	Status         BlobStatus     `gorm:"status" json:"status"`
	Version        int            `gorm:"version" json:"version"`
	CreatedAt      time.Time      `gorm:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// BeforeCreate hook for BlobModel to add a prefixed UUID
func (blob *BlobModel) BeforeCreate(tx *gorm.DB) (err error) {
	blob.ID = uuid.New()
	return
}
