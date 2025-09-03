package domain

import (
	"context"
	"fmt"
	"io"
)

type FileBackend interface {
	Name() string

	// Put writes/overwrites a file.
	Put(path string, data []byte) error

	// Get reads the entire file into memory.
	Get(path string) ([]byte, error)

	// Stream returns a reader for the file contents. Caller must Close().
	Stream(path string) (io.ReadCloser, error)

	// Delete removes the file.
	Delete(path string) error

	// Exists checks if the file exists.
	Exists(path string) (bool, error)

	// Stat returns metadata like size, modified time, etag, etc.
	// Stat(path string) (*model.FileInfo, error)

	// DeleteFolder removes all objects with the given prefix.
	// For local backend, this will simply remove the directory.
	DeleteFolder(prefix string) error

	Move(oldPath, newPath string) error
}

type MigratableBackend interface {
	FileBackend

	// Kicks off a background migration of all existing files
	MigrateAll(ctx context.Context) error

	// Migrate a specific file (used for lazy migration on access)
	MigrateFile(ctx context.Context, path string) error

	// Progress info for observability
	MigrationStatus() (completed int64, total int64)
}

type PlaceholderBackend struct {
	Title string
}

var _ FileBackend = (*PlaceholderBackend)(nil)

func (p *PlaceholderBackend) Name() string { return p.Title }

// Every other method should return an error, not panic
func (p *PlaceholderBackend) Put(path string, data []byte) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}
func (p *PlaceholderBackend) Get(path string) ([]byte, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Delete implements domain.FileBackend.
func (p *PlaceholderBackend) Delete(path string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// DeleteFolder implements domain.FileBackend.
func (p *PlaceholderBackend) DeleteFolder(prefix string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Exists implements domain.FileBackend.
func (p *PlaceholderBackend) Exists(path string) (bool, error) {
	return false, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Move implements domain.FileBackend.
func (p *PlaceholderBackend) Move(oldPath string, newPath string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Stream implements domain.FileBackend.
func (p *PlaceholderBackend) Stream(path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}
