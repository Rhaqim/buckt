package backend

import (
	"context"
	"fmt"
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
)

type MigrationBackendService struct {
	logger domain.BucktLogger

	primaryBackend   domain.FileBackend
	secondaryBackend domain.FileBackend

	// migrating atomic.Bool // indicates an active migration
	// stats     migrationStats
}

func NewMigrationBackend(bucktLogger domain.BucktLogger, primary domain.FileBackend, secondary domain.FileBackend) domain.MigratableBackend {
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
func (d *MigrationBackendService) Put(ctx context.Context, path string, data []byte) error {
	// Try to put the file in the primary backend
	if err := d.primaryBackend.Put(ctx, path, data); err != nil {
		d.logger.Errorf("Failed to put file in primary backend: %v", err)
	}

	// If the primary backend fails, try the secondary backend
	if err := d.secondaryBackend.Put(ctx, path, data); err != nil {
		d.logger.Errorf("⚠️ Failed to mirror to secondary: %v", err)
		return err
	}
	return nil
}

// Get implements domain.FileBackend.
func (d *MigrationBackendService) Get(ctx context.Context, path string) ([]byte, error) {
	// Try to get the file from the primary backend
	data, err := d.primaryBackend.Get(ctx, path)
	if err != nil {
		d.logger.Errorf("Failed to get file from primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		data, err = d.secondaryBackend.Get(ctx, path)
		if err != nil {
			d.logger.Errorf("Failed to get file from secondary backend: %v", err)
			return nil, err
		}
	}
	return data, nil
}

// List implements domain.FileBackend.
func (d *MigrationBackendService) List(ctx context.Context, prefix string) ([]string, error) {
	// Try to list files from the primary backend
	paths, err := d.primaryBackend.List(ctx, prefix)
	if err != nil {
		d.logger.Errorf("Failed to list files from primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		paths, err = d.secondaryBackend.List(ctx, prefix)
		if err != nil {
			d.logger.Errorf("Failed to list files from secondary backend: %v", err)
			return nil, err
		}
	}
	return paths, nil
}

// Stream implements domain.FileBackend.
func (d *MigrationBackendService) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	// Try to stream the file from the primary backend
	reader, err := d.primaryBackend.Stream(ctx, path)
	if err != nil {
		d.logger.Errorf("Failed to stream file from primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		reader, err = d.secondaryBackend.Stream(ctx, path)
		if err != nil {
			d.logger.Errorf("Failed to stream file from secondary backend: %v", err)
			return nil, err
		}
	}
	return reader, nil
}

// Move implements domain.FileBackend.
func (d *MigrationBackendService) Move(ctx context.Context, oldPath string, newPath string) error {
	// Try to move the file in the primary backend
	if err := d.primaryBackend.Move(ctx, oldPath, newPath); err != nil {
		d.logger.Errorf("Failed to move file in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.Move(ctx, oldPath, newPath); err != nil {
			d.logger.Errorf("Failed to move file in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// Exists implements domain.FileBackend.
func (d *MigrationBackendService) Exists(ctx context.Context, path string) (bool, error) {
	// Check if the file exists in the primary backend
	exists, err := d.primaryBackend.Exists(ctx, path)
	if err != nil {
		d.logger.Errorf("Failed to check existence in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		exists, err = d.secondaryBackend.Exists(ctx, path)
		if err != nil {
			d.logger.Errorf("Failed to check existence in secondary backend: %v", err)
			return false, err
		}
	}
	return exists, nil
}

// Delete implements domain.FileBackend.
func (d *MigrationBackendService) Delete(ctx context.Context, path string) error {
	// Try to delete the file in the primary backend
	if err := d.primaryBackend.Delete(ctx, path); err != nil {
		d.logger.Errorf("Failed to delete file in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.Delete(ctx, path); err != nil {
			d.logger.Errorf("Failed to delete file in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// DeleteFolder implements domain.FileBackend.
func (d *MigrationBackendService) DeleteFolder(ctx context.Context, prefix string) error {
	// Try to delete the folder in the primary backend
	if err := d.primaryBackend.DeleteFolder(ctx, prefix); err != nil {
		d.logger.Errorf("Failed to delete folder in primary backend: %v", err)
		// If the primary backend fails, try the secondary backend
		if err := d.secondaryBackend.DeleteFolder(ctx, prefix); err != nil {
			d.logger.Errorf("Failed to delete folder in secondary backend: %v", err)
			return err
		}
	}
	return nil
}

// MigrateAll implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrateAll(ctx context.Context) error {
	return fmt.Errorf("MigrateAll not implemented")
}

// MigrateFile implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrateFile(ctx context.Context, path string) error {
	return fmt.Errorf("MigrateFile not implemented")
}

// MigrationStatus implements domain.MigratableBackend.
func (d *MigrationBackendService) MigrationStatus(ctx context.Context) (completed int64, total int64) {
	return 0, 0
}
