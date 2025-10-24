package mocks

import (
	"context"
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
)

type Backend struct {
	NameVal string
}

var _ domain.FileBackend = (*Backend)(nil)

// Delete implements domain.FileBackend.
func (b *Backend) Delete(ctx context.Context, path string) error {
	return nil
}

// DeleteFolder implements domain.FileBackend.
func (b *Backend) DeleteFolder(ctx context.Context, prefix string) error {
	return nil
}

// Exists implements domain.FileBackend.
func (b *Backend) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}

// Get implements domain.FileBackend.
func (b *Backend) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

// Move implements domain.FileBackend.
func (b *Backend) Move(ctx context.Context, oldPath string, newPath string) error {
	return nil
}

// Name implements domain.FileBackend.
func (b *Backend) Name() string {
	return b.NameVal
}

// Put implements domain.FileBackend.
func (b *Backend) Put(ctx context.Context, path string, data []byte) error {
	return nil
}

// Stream implements domain.FileBackend.
func (b *Backend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}

type MigrationBackend struct {
	Backend
}

var _ domain.MigratableBackend = (*MigrationBackend)(nil)

// MigrateAll implements domain.MigratableBackend.
func (m *MigrationBackend) MigrateAll(ctx context.Context) error {
	return nil
}

// MigrateFile implements domain.MigratableBackend.
func (m *MigrationBackend) MigrateFile(ctx context.Context, path string) error {
	return nil
}

// MigrationStatus implements domain.MigratableBackend.
func (m *MigrationBackend) MigrationStatus(ctx context.Context) (completed int64, total int64) {
	return 0, 0
}
