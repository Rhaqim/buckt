package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderModel struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"` // Unique identifier for the file
	Name     string    `gorm:"not null"`             // File name
	BucketID uuid.UUID `gorm:"type:uuid;not null"`   // Foreign key to BucketModel
	ParentID uuid.UUID `gorm:"type:uuid;not null"`   // Foreign key to FolderModel
	gorm.Model
}

type FolderRepository struct {
	db *gorm.DB
}

func NewFolderRepository(db *gorm.DB) domain.Repository[FolderModel] {
	return &FolderRepository{db}
}

func (r *FolderRepository) Create(file *FolderModel) error {
	return r.db.Create(file).Error
}

func (r *FolderRepository) FindAll() ([]FolderModel, error) {
	var files []FolderModel
	err := r.db.Find(&files).Error
	return files, err
}

func (r *FolderRepository) FindByID(id uuid.UUID) (FolderModel, error) {
	var file FolderModel
	err := r.db.First(&file, id).Error
	return file, err
}

func (r *FolderRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&FolderModel{}, id).Error
}

func (r *FolderRepository) GetBy(key string, value string) (FolderModel, error) {
	var file FolderModel
	err := r.db.Where(key+" = ?", value).First(&file).Error
	return file, err
}

func (r *FolderRepository) GetMany(key string, value string) ([]FolderModel, error) {
	var files []FolderModel

	err := r.db.Where(key+" = ?", value).Find(&files).Error
	return files, err
}

// BeforeCreate hook for FolderModel to add a prefixed UUID
func (folder *FolderModel) BeforeCreate(tx *gorm.DB) (err error) {
	folder.ID = uuid.New()
	return
}
