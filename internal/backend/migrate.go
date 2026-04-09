package backend

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/Rhaqim/buckt/internal/domain"
)

type MigrationBackendService struct {
	logger domain.BucktLogger

	primaryBackend   domain.FileBackend
	secondaryBackend domain.FileBackend

	migrating atomic.Bool
	stats     migrationStats
}

type migrationStats struct {
	mu        sync.Mutex
	completed int64
	total     int64
}

func NewMigrationBackend(bucktLogger domain.BucktLogger, primary domain.FileBackend, secondary domain.FileBackend) domain.MigratableBackend {
	bucktLogger.Info("🚀 Initialising migration backend: " + primary.Name() + " -> " + secondary.Name())
	return &MigrationBackendService{
		logger:           bucktLogger,
		primaryBackend:   primary,
		secondaryBackend: secondary,
	}
}

func (d *MigrationBackendService) Name() string {
	return d.primaryBackend.Name() + "->" + d.secondaryBackend.Name()
}

// Put writes to both backends. Primary failure is a hard error.
// Secondary failure is logged but does not fail the operation.
func (d *MigrationBackendService) Put(ctx context.Context, path string, data []byte) error {
	// Write to primary first — this is the source of truth
	if err := d.primaryBackend.Put(ctx, path, data); err != nil {
		return fmt.Errorf("primary backend put failed: %w", err)
	}

	// Mirror to secondary (best-effort during migration)
	if err := d.secondaryBackend.Put(ctx, path, data); err != nil {
		d.logger.Errorf("⚠️ Failed to mirror to secondary backend: %v", err)
	}
	return nil
}

// Get reads from primary first, falls back to secondary.
func (d *MigrationBackendService) Get(ctx context.Context, path string) ([]byte, error) {
	data, err := d.primaryBackend.Get(ctx, path)
	if err != nil {
		d.logger.Errorf("Primary backend get failed, trying secondary: %v", err)
		data, err = d.secondaryBackend.Get(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("both backends failed to get %s: %w", path, err)
		}
		// Lazy migration: copy to primary since it was missing
		if putErr := d.primaryBackend.Put(ctx, path, data); putErr != nil {
			d.logger.Errorf("⚠️ Failed to lazy-migrate %s to primary: %v", path, putErr)
		}
	}
	return data, nil
}

// List lists from primary first, falls back to secondary.
func (d *MigrationBackendService) List(ctx context.Context, prefix string) ([]string, error) {
	paths, err := d.primaryBackend.List(ctx, prefix)
	if err != nil {
		d.logger.Errorf("Primary backend list failed, trying secondary: %v", err)
		paths, err = d.secondaryBackend.List(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("both backends failed to list %s: %w", prefix, err)
		}
	}
	return paths, nil
}

// Stream streams from primary first, falls back to secondary.
func (d *MigrationBackendService) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	reader, err := d.primaryBackend.Stream(ctx, path)
	if err != nil {
		d.logger.Errorf("Primary backend stream failed, trying secondary: %v", err)
		reader, err = d.secondaryBackend.Stream(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("both backends failed to stream %s: %w", path, err)
		}
	}
	return reader, nil
}

// Move moves in both backends.
func (d *MigrationBackendService) Move(ctx context.Context, oldPath string, newPath string) error {
	if err := d.primaryBackend.Move(ctx, oldPath, newPath); err != nil {
		return fmt.Errorf("primary backend move failed: %w", err)
	}
	if err := d.secondaryBackend.Move(ctx, oldPath, newPath); err != nil {
		d.logger.Errorf("⚠️ Failed to move in secondary backend: %v", err)
	}
	return nil
}

// Exists checks primary first, then secondary.
func (d *MigrationBackendService) Exists(ctx context.Context, path string) (bool, error) {
	exists, err := d.primaryBackend.Exists(ctx, path)
	if err != nil {
		d.logger.Errorf("Primary backend exists check failed, trying secondary: %v", err)
		exists, err = d.secondaryBackend.Exists(ctx, path)
		if err != nil {
			return false, fmt.Errorf("both backends failed exists check for %s: %w", path, err)
		}
	}
	return exists, nil
}

// Delete deletes from both backends. Primary failure is a hard error.
func (d *MigrationBackendService) Delete(ctx context.Context, path string) error {
	if err := d.primaryBackend.Delete(ctx, path); err != nil {
		return fmt.Errorf("primary backend delete failed: %w", err)
	}
	if err := d.secondaryBackend.Delete(ctx, path); err != nil {
		d.logger.Errorf("⚠️ Failed to delete from secondary backend: %v", err)
	}
	return nil
}

// DeleteFolder deletes from both backends.
func (d *MigrationBackendService) DeleteFolder(ctx context.Context, prefix string) error {
	if err := d.primaryBackend.DeleteFolder(ctx, prefix); err != nil {
		return fmt.Errorf("primary backend delete folder failed: %w", err)
	}
	if err := d.secondaryBackend.DeleteFolder(ctx, prefix); err != nil {
		d.logger.Errorf("⚠️ Failed to delete folder from secondary backend: %v", err)
	}
	return nil
}

// MigrateFile copies a single file from primary to secondary.
func (d *MigrationBackendService) MigrateFile(ctx context.Context, path string) error {
	data, err := d.primaryBackend.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read %s from primary: %w", path, err)
	}

	if err := d.secondaryBackend.Put(ctx, path, data); err != nil {
		return fmt.Errorf("failed to write %s to secondary: %w", path, err)
	}

	d.stats.mu.Lock()
	d.stats.completed++
	d.stats.mu.Unlock()

	return nil
}

// MigrateAll copies all files from primary to secondary in the background.
func (d *MigrationBackendService) MigrateAll(ctx context.Context) error {
	if d.migrating.Load() {
		return fmt.Errorf("migration already in progress")
	}
	d.migrating.Store(true)

	// List all files from primary
	files, err := d.primaryBackend.List(ctx, "")
	if err != nil {
		d.migrating.Store(false)
		return fmt.Errorf("failed to list files from primary: %w", err)
	}

	d.stats.mu.Lock()
	d.stats.total = int64(len(files))
	d.stats.completed = 0
	d.stats.mu.Unlock()

	// Run migration in a goroutine
	go func() {
		defer d.migrating.Store(false)

		for _, path := range files {
			select {
			case <-ctx.Done():
				d.logger.Errorf("Migration cancelled: %v", ctx.Err())
				return
			default:
			}

			// Skip if already exists in secondary
			exists, err := d.secondaryBackend.Exists(ctx, path)
			if err == nil && exists {
				d.stats.mu.Lock()
				d.stats.completed++
				d.stats.mu.Unlock()
				continue
			}

			if err := d.MigrateFile(ctx, path); err != nil {
				d.logger.Errorf("Failed to migrate %s: %v", path, err)
				// Continue with other files
			}
		}

		d.logger.Info("✅ Migration complete")
	}()

	return nil
}

// MigrationStatus returns the current migration progress.
func (d *MigrationBackendService) MigrationStatus(ctx context.Context) (completed int64, total int64) {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()
	return d.stats.completed, d.stats.total
}
