package domain

import (
	"context"
	"fmt"
	"io"
)

type FileBackend interface {
	Name() string

	// Put writes/overwrites a file.
	Put(ctx context.Context, path string, data []byte) error

	// Get reads the entire file into memory.
	Get(ctx context.Context, path string) ([]byte, error)

	// Stream returns a reader for the file contents. Caller must Close().
	Stream(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes the file.
	Delete(ctx context.Context, path string) error

	// Exists checks if the file exists.
	Exists(ctx context.Context, path string) (bool, error)

	// Stat returns metadata like size, modified time, etag, etc.
	// Stat(path string) (*model.FileInfo, error)

	// DeleteFolder removes all objects with the given prefix.
	// For local backend, this will simply remove the directory.
	DeleteFolder(ctx context.Context, prefix string) error

	Move(ctx context.Context, oldPath, newPath string) error
}

type MigratableBackend interface {
	FileBackend

	// Kicks off a background migration of all existing files
	MigrateAll(ctx context.Context) error

	// Migrate a specific file (used for lazy migration on access)
	MigrateFile(ctx context.Context, path string) error

	// Progress info for observability
	MigrationStatus(ctx context.Context) (completed int64, total int64)
}

type PlaceholderBackend struct {
	Title string
}

var _ FileBackend = (*PlaceholderBackend)(nil)

func (p *PlaceholderBackend) Name() string { return p.Title }

// Every other method should return an error, not panic
func (p *PlaceholderBackend) Put(ctx context.Context, path string, data []byte) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}
func (p *PlaceholderBackend) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Delete implements domain.FileBackend.
func (p *PlaceholderBackend) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// DeleteFolder implements domain.FileBackend.
func (p *PlaceholderBackend) DeleteFolder(ctx context.Context, prefix string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Exists implements domain.FileBackend.
func (p *PlaceholderBackend) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Move implements domain.FileBackend.
func (p *PlaceholderBackend) Move(ctx context.Context, oldPath string, newPath string) error {
	return fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Stream implements domain.FileBackend.
func (p *PlaceholderBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// BucktFileSystemService defines the interface for file system operations within the Buckt domain.
// It provides methods to validate paths, write, retrieve, update, and delete files.
type FileSystemService interface {
	// FSValidatePath validates the given file path and returns the validated path or an error.
	FSValidatePath(path string) (string, error)

	// FSWriteFile writes the given file data to the specified path.
	// Returns an error if the operation fails.
	FSWriteFile(path string, file []byte) error

	// FSGetFile retrieves the file data from the specified path.
	// Returns the file data or an error if the operation fails.
	FSGetFile(path string) ([]byte, error)

	// FSGetFileStream retrieves the file data from the specified path.
	// Returns an io.ReadCloser or an error if the operation fails.
	FSGetFileStream(path string) (io.ReadCloser, error)

	// FSUpdateFile updates the file from the old path to the new path.
	// Returns an error if the operation fails.
	FSUpdateFile(oldPath, newPath string) error

	// FSDeleteFile deletes the file or folder at the specified path.
	// Returns an error if the operation fails.
	FSDeleteFile(folderPath string) error

	// FSDeleteFolder deletes the folder at the specified path.
	// Returns an error if the operation fails.
	FSDeleteFolder(folderPath string) error
}
