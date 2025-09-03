package backend

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type MigrationBackendService struct {
	logger *logger.BucktLogger

	primaryBackend   domain.FileBackend
	secondaryBackend domain.FileBackend

	migrating atomic.Bool // indicates an active migration
	// stats     migrationStats
}

func NewMigrationBackend(bucktLogger *logger.BucktLogger, primary domain.FileBackend, secondary domain.FileBackend) domain.MigratableBackend {
	bucktLogger.Info("🚀 Initialising local file system backend")
	return &MigrationBackendService{
		logger:           bucktLogger,
		primaryBackend:   primary,
		secondaryBackend: secondary,
	}
}

func (d *MigrationBackendService) Name() string {
	return d.primaryBackend.Name() + "->" + d.secondaryBackend.Name()
}

// Put implements domain.FileBackend.
func (d *MigrationBackendService) Put(path string, data []byte) error {
	// Try to put the file in the primary backend
	if err := d.primaryBackend.Put(path, data); err != nil {
		d.logger.Errorf("Failed to put file in primary backend: %v", err)
	}

	// If the primary backend fails, try the secondary backend
	if err := d.secondaryBackend.Put(path, data); err != nil {
		d.logger.Errorf("⚠️ Failed to mirror to secondary: %v", err)
		return err
	}
	return nil
}

// Get implements domain.FileBackend.
func (d *MigrationBackendService) Get(path string) ([]byte, error) {
	// Try to get the file from the primary backend
	data, err := d.primaryBackend.Get(path)
	if err != nil {
		d.logger.Errorf("Failed to get file from primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		data, err = d.secondaryBackend.Get(path)
		if err != nil {
			d.logger.Errorf("Failed to get file from secondary backend: %v", err)
			return nil, err
		}
	}
	return data, nil
}

// Stream implements domain.FileBackend.
func (d *MigrationBackendService) Stream(path string) (io.ReadCloser, error) {
	// Try to stream the file from the primary backend
	reader, err := d.primaryBackend.Stream(path)
	if err != nil {
		d.logger.Errorf("Failed to stream file from primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		reader, err = d.secondaryBackend.Stream(path)
		if err != nil {
			d.logger.Errorf("Failed to stream file from secondary backend: %v", err)
			return nil, err
		}
	}
	return reader, nil
}

// Move implements domain.FileBackend.
func (d *MigrationBackendService) Move(oldPath string, newPath string) error {
	// Try to move the file in the primary backend
	if err := d.primaryBackend.Move(oldPath, newPath); err != nil {
		d.logger.Errorf("Failed to move file in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.Move(oldPath, newPath); err != nil {
			d.logger.Errorf("Failed to move file in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// Exists implements domain.FileBackend.
func (d *MigrationBackendService) Exists(path string) (bool, error) {
	// Check if the file exists in the primary backend
	exists, err := d.primaryBackend.Exists(path)
	if err != nil {
		d.logger.Errorf("Failed to check existence in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		exists, err = d.secondaryBackend.Exists(path)
		if err != nil {
			d.logger.Errorf("Failed to check existence in secondary backend: %v", err)
			return false, err
		}
	}
	return exists, nil
}

// Delete implements domain.FileBackend.
func (d *MigrationBackendService) Delete(path string) error {
	// Try to delete the file in the primary backend
	if err := d.primaryBackend.Delete(path); err != nil {
		d.logger.Errorf("Failed to delete file in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.Delete(path); err != nil {
			d.logger.Errorf("Failed to delete file in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// DeleteFolder implements domain.FileBackend.
func (d *MigrationBackendService) DeleteFolder(prefix string) error {
	// Try to delete the folder in the primary backend
	if err := d.primaryBackend.DeleteFolder(prefix); err != nil {
		d.logger.Errorf("Failed to delete folder in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.DeleteFolder(prefix); err != nil {
			d.logger.Errorf("Failed to delete folder in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// MigrateAll implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrateAll(ctx context.Context) error {
	panic("unimplemented")
}

// MigrateFile implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrateFile(ctx context.Context, path string) error {
	panic("unimplemented")
}

// MigrationStatus implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrationStatus() (completed int64, total int64) {
	panic("unimplemented")
}
