package repository

import (
	"context"

	"github.com/Rhaqim/buckt/internal/domain"
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

// NewMigrationStateStore returns a domain.MigrationStateStore backed by the
// MigrationModel table, used to persist bulk-migration progress for resume.
func NewMigrationStateStore(db *gorm.DB) domain.MigrationStateStore {
	return &migrationRepository{db: db}
}

// MigratedKeys implements domain.MigrationStateStore.
func (repo *migrationRepository) MigratedKeys(ctx context.Context, backend string) (map[string]struct{}, error) {
	var keys []string
	err := repo.db.WithContext(ctx).
		Model(&model.MigrationModel{}).
		Where("backend = ? AND status = ?", backend, model.MigrationStatusCommitted).
		Pluck("object_key", &keys).Error
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set, nil
}

// MarkMigrated implements domain.MigrationStateStore. It is idempotent: an
// existing (backend, key) row is left untouched rather than duplicated.
func (repo *migrationRepository) MarkMigrated(ctx context.Context, backend, key string, size int64) error {
	var m model.MigrationModel
	return repo.db.WithContext(ctx).
		Where("backend = ? AND object_key = ?", backend, key).
		Attrs(model.MigrationModel{
			Backend:   model.MigrationBackend(backend),
			ObjectKey: key,
			Size:      size,
			Status:    model.MigrationStatusCommitted,
		}).
		FirstOrCreate(&m).Error
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
