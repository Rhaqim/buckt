package model

import (
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OwnerModel struct {
	ID      uuid.UUID     `gorm:"type:uuid;primaryKey"` // Unique identifier for the owner
	Name    string        `gorm:"not null"`             // Owner name
	Email   string        `gorm:"not null;unique"`      // Owner email
	Buckets []BucketModel `gorm:"foreignKey:OwnerID"`   // Establish one-to-many relationship with BucketModel
}

type OwnerRepository struct {
	db *gorm.DB
}

func NewOwnerRepository(db *gorm.DB) domain.Repository[OwnerModel] {
	return &OwnerRepository{db}
}

func (r *OwnerRepository) Create(owner *OwnerModel) error {
	return r.db.Create(owner).Error
}

func (r *OwnerRepository) FindAll() ([]OwnerModel, error) {
	var owners []OwnerModel
	err := r.db.Find(&owners).Error
	return owners, err
}

func (r *OwnerRepository) FindByID(id uuid.UUID) (OwnerModel, error) {
	var owner OwnerModel
	err := r.db.First(&owner, id).Error
	return owner, err
}

func (r *OwnerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&OwnerModel{}, id).Error
}

func (r *OwnerRepository) GetBy(key string, value string) (OwnerModel, error) {
	var owner OwnerModel

	err := r.db.Where(key+" = ?", value).First(&owner).Error
	return owner, err
}

// BeforeCreate hook for OwnerModel to add a prefixed UUID
func (owner *OwnerModel) BeforeCreate(tx *gorm.DB) (err error) {
	owner.ID = uuid.New()
	return
}
