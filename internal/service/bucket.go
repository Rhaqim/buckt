package service

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/google/uuid"
)

type BucketService struct {
	*logger.Logger

	domain.BucketRepository
}

func NewBucketService(log *logger.Logger, bucketRepository domain.BucketRepository) domain.BucketService {
	return &BucketService{
		Logger:           log,
		BucketRepository: bucketRepository,
	}
}

// CreateBucket implements domain.BucketService.
func (b *BucketService) CreateBucket(user_id string, bukcet_name string, description string) error {
	bucket := &model.BucketModel{
		UserID:      user_id,
		Name:        bukcet_name,
		Description: description,
	}

	if err := b.BucketRepository.Create(bucket); err != nil {
		return b.WrapError("failed to create bucket", err)
	}

	return nil
}

// GetBucket implements domain.BucketService.
// Subtle: this method shadows the method (BucketRepository).GetBucket of BucketService.BucketRepository.
func (b *BucketService) GetBucket(user_id string, bucket_id string) (*model.BucketModel, error) {
	id, err := uuid.Parse(bucket_id)
	if err != nil {
		return nil, b.WrapError("failed to parse uuid", err)
	}

	bucket, err := b.BucketRepository.GetBucket(user_id, id)
	if err != nil {
		return nil, b.WrapError("failed to get bucket", err)
	}

	return bucket, nil
}

// GetBuckets implements domain.BucketService.
// Subtle: this method shadows the method (BucketRepository).GetBuckets of BucketService.BucketRepository.
func (b *BucketService) GetBuckets(user_id string) ([]model.BucketModel, error) {
	buckets, err := b.BucketRepository.GetBuckets(user_id)
	if err != nil {
		return nil, b.WrapError("failed to get buckets", err)
	}

	return buckets, nil
}
