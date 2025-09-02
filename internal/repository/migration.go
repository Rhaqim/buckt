package repository

import (
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MigrationRepository interface {
	CreateMigration(migration *model.MigrationModel) error
	GetMigration(id uuid.UUID) (*model.MigrationModel, error)
	UpdateMigration(migration *model.MigrationModel) error
	DeleteMigration(id uuid.UUID) error
}

type migrationRepository struct {
	db *gorm.DB
}

func NewMigrationRepository(db *gorm.DB) MigrationRepository {
	return &migrationRepository{
		db: db,
	}
}

func (repo *migrationRepository) CreateMigration(migration *model.MigrationModel) error {
	return repo.db.Create(migration).Error
}

func (repo *migrationRepository) GetMigration(id uuid.UUID) (*model.MigrationModel, error) {
	var migration model.MigrationModel
	err := repo.db.First(&migration, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &migration, nil
}

func (repo *migrationRepository) UpdateMigration(migration *model.MigrationModel) error {
	return repo.db.Save(migration).Error
}

func (repo *migrationRepository) DeleteMigration(id uuid.UUID) error {
	return repo.db.Delete(&model.MigrationModel{}, id).Error
}
