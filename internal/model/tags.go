package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TagModel struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"` // Unique identifier for the tag
	Name string    `gorm:"not null"`             // Tag name
	gorm.Model
}

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) domain.BucktRepository[TagModel] {
	return &TagRepository{db}
}

func (r *TagRepository) Create(tag *TagModel) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) Update(tag *TagModel) error {
	return r.db.Save(tag).Error
}

func (r *TagRepository) GetAll() ([]TagModel, error) {
	var tags []TagModel
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *TagRepository) GetByID(id uuid.UUID) (TagModel, error) {
	var tag TagModel
	err := r.db.First(&tag, id).Error
	return tag, err
}

func (r *TagRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&TagModel{}, id).Error
}

func (r *TagRepository) GetBy(key interface{}, value ...interface{}) (TagModel, error) {
	var tag TagModel

	err := r.db.Where(key, value...).First(&tag).Error
	return tag, err
}

func (r *TagRepository) GetMany(key interface{}, value ...interface{}) ([]TagModel, error) {
	var tags []TagModel
	err := r.db.Where(key, value).Find(&tags).Error
	return tags, err
}

func (r *TagRepository) RawQuery(query string, values ...interface{}) *gorm.DB {
	return r.db.Raw(query, values)
}

// BeforeCreate hook for TagModel to add a prefixed UUID
func (tag *TagModel) BeforeCreate(tx *gorm.DB) (err error) {
	tag.ID = uuid.New()
	return
}
