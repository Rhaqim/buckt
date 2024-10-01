package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"gorm.io/gorm"
)

type BucketModel struct {
	gorm.Model
	Name string
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

func (r *BucketRepository) FindByID(id uint) (BucketModel, error) {
	var file BucketModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *BucketRepository) Delete(id uint) error {
	return r.db.Delete(&BucketModel{}, id).Error
}

func (r *BucketRepository) GetBy(key string, value string) (BucketModel, error) {
	var file BucketModel

	err := r.db.Where(key+" = ?", value).First(&file).Error
	return file, err
}
