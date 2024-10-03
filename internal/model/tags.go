package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TagModel struct {
	ID   string `gorm:"type:uuid;primaryKey"` // Unique identifier for the tag
	Name string `gorm:"not null"`             // Tag name
	gorm.Model
}

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) domain.Repository[TagModel] {
	return &TagRepository{db}
}

func (r *TagRepository) Create(tag *TagModel) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) FindAll() ([]TagModel, error) {
	var tags []TagModel
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *TagRepository) FindByID(id string) (TagModel, error) {
	var tag TagModel
	err := r.db.First(&tag, id).Error
	return tag, err
}

func (r *TagRepository) Delete(id string) error {
	return r.db.Delete(&TagModel{}, id).Error
}

func (r *TagRepository) GetBy(key string, value string) (TagModel, error) {
	var tag TagModel

	err := r.db.Where(key+" = ?", value).First(&tag).Error
	return tag, err
}

// BeforeCreate hook for TagModel to add a prefixed UUID
func (tag *TagModel) BeforeCreate(tx *gorm.DB) (err error) {
	tag.ID = "tag-" + uuid.New().String()
	return
}
