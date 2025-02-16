package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BucketModel struct {
	gorm.Model
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey"` // Unique identifier for the bucket
	UserID      string        `gorm:"not null"`             // ID of the user who owns the bucket
	Description string        `gorm:"type:text"`            // Optional description of the bucket
	Name        string        `gorm:"unique;not null"`      // Ensure each bucket name is unique
	Folders     []FolderModel `gorm:"foreignKey:BucketID"`  // Establish one-to-many relationship with FolderModel
	Files       []FileModel   `gorm:"foreignKey:ParentID"`  // Establish one-to-many relationship with FileModel
}

// BeforeCreate hook for BucketModel to add a prefixed UUID
func (bucket *BucketModel) BeforeCreate(tx *gorm.DB) (err error) {
	bucket.ID = uuid.New()
	return
}

// type BucketRepository struct {
// 	db *gorm.DB
// }

// func NewBucketRepository(db *gorm.DB) domain_old.BucktRepository[BucketModel] {
// 	return &BucketRepository{db}
// }

// func (r *BucketRepository) Create(file *BucketModel) error {
// 	return r.db.Create(file).Error
// }

// func (r *BucketRepository) Update(file *BucketModel) error {
// 	return r.db.Save(file).Error
// }

// func (r *BucketRepository) GetAll() ([]BucketModel, error) {
// 	var files []BucketModel
// 	err := r.db.Find(&files).Error
// 	return files, err
// }

// func (r *BucketRepository) GetByID(id uuid.UUID) (BucketModel, error) {
// 	var file BucketModel
// 	err := r.db.First(&file, id).Error
// 	return file, err
// }

// func (r *BucketRepository) Delete(id uuid.UUID) error {
// 	return r.db.Delete(&BucketModel{}, id).Error
// }

// func (r *BucketRepository) GetBy(key interface{}, value ...interface{}) (BucketModel, error) {
// 	var file BucketModel

// 	err := r.db.Where(key, value...).First(&file).Error
// 	return file, err
// }

// func (r *BucketRepository) GetMany(key interface{}, value ...interface{}) ([]BucketModel, error) {
// 	var files []BucketModel

// 	err := r.db.Where(key, value).Find(&files).Error

// 	return files, err
// }

// func (r *BucketRepository) RawQuery(query string, values ...interface{}) *gorm.DB {
// 	return r.db.Raw(query, values...)
// }
