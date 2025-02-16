package repository

import (
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type BucketRepository struct {
	*database.DB
	*logger.Logger
}

func NewBucketRepository(db *database.DB, logger *logger.Logger) domain.BucketRepository {
	return &BucketRepository{db, logger}
}

// Create implements domain.BucketRepository.
// Subtle: this method shadows the method (*DB).Create of BucketRepository.DB.
func (b *BucketRepository) Create(bucket *model.BucketModel) error {
	return b.DB.Create(bucket).Error
}

// GetUserBucket implements domain.BucketRepository.
func (b *BucketRepository) GetBucket(user_id string, bucket_id uuid.UUID) (*model.BucketModel, error) {
	var bucket model.BucketModel
	err := b.DB.Preload("Folders").Preload("Files").Where("user_id = ? AND id = ?", user_id, bucket_id).First(&bucket).Error
	return &bucket, err
}

// GetBuckets implements domain.BucketRepository.
func (b *BucketRepository) GetBuckets(user_id string) ([]model.BucketModel, error) {
	var buckets []model.BucketModel
	err := b.DB.Preload("Folders").Preload("Files").Where("user_id = ?", user_id).Find(&buckets).Error
	return buckets, err
}
