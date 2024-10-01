package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BucketModel struct {
	ID          string      `gorm:"type:uuid;primaryKey"` // Unique identifier for the bucket
	Name        string      `gorm:"unique;not null"`      // Ensure each bucket name is unique
	Description string      `gorm:"type:text"`            // Optional description of the bucket
	Files       []FileModel `gorm:"foreignKey:BucketID"`  // Establish one-to-many relationship with FileModel
	gorm.Model
}

type BucketRepository struct {
	db *gorm.DB
}

func NewBucketRepository(db *gorm.DB) domain.Repository[BucketModel] {
	return &BucketRepository{db}
}

func (r *BucketRepository) Create(file *BucketModel) error {
	return r.db.Create(file).Error
}

func (r *BucketRepository) FindAll() ([]BucketModel, error) {
	var files []BucketModel
	err := r.db.Find(&files).Error
	return files, err
}

func (r *BucketRepository) FindByID(id string) (BucketModel, error) {
	var file BucketModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *BucketRepository) Delete(id string) error {
	return r.db.Delete(&BucketModel{}, id).Error
}

func (r *BucketRepository) GetBy(key string, value string) (BucketModel, error) {
	var file BucketModel

	err := r.db.Where(key+" = ?", value).First(&file).Error
	return file, err
}

// BeforeCreate hook for BucketModel to add a prefixed UUID
func (bucket *BucketModel) BeforeCreate(tx *gorm.DB) (err error) {
	bucket.ID = "bucket-" + uuid.New().String()
	return
}
