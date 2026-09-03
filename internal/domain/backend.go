package domain

import (
	"context"
	"fmt"
	"io"
	"time"
)

type FileBackend interface {
	Name() string

	// Put writes/overwrites a file.
	Put(ctx context.Context, path string, data []byte) error

	// List(prefix string) ([]string, error)
	List(ctx context.Context, prefix string) ([]string, error)

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

// PresignBackend is an OPTIONAL capability: a backend that can mint a
// time-limited URL granting direct access to an object, so reads don't have to
// stream through the application process. Cloud backends (S3/R2, and later
// Azure/GCS) implement it; the local filesystem backend does not. buckt detects
// it by type assertion (like MigratableBackend) and surfaces ErrUnsupported when
// the active backend can't presign.
type PresignBackend interface {
	// PresignGetURL returns a URL that downloads the object at key directly from
	// the backend, valid for ttl. The URL bypasses buckt's auth for its lifetime,
	// so keep ttl short.
	PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)

	// PresignPutURL returns a URL that uploads (HTTP PUT) an object to key
	// directly to the backend, valid for ttl — so a client can upload straight to
	// storage without the bytes passing through the application. Used by the
	// register/confirm upload flow (CreatePendingUpload / FinalizeUpload).
	PresignPutURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type MigratableBackend interface {
	FileBackend

	// Kicks off a background migration of all existing files
	MigrateAll(ctx context.Context) error

	// Migrate a specific file (used for lazy migration on access)
	MigrateFile(ctx context.Context, path string) error

	// Progress info for observability. completed counts files copied (or already
	// present); failed counts files that permanently failed after retries;
	// total is the number scheduled. completed+failed == total when done.
	MigrationStatus(ctx context.Context) (completed int64, failed int64, total int64)
}

// MigrationStateStore persists which object keys have been copied to a target
// backend, so a bulk migration can resume across a restart without re-scanning
// the target. Keyed by the target backend's Name(). Implementations must be
// safe for concurrent use. Persistence is best-effort from the migration's
// point of view: a failed record never fails a copy (the copy is idempotent, so
// the worst case is re-copying the object on the next run).
type MigrationStateStore interface {
	// MigratedKeys returns the set of keys already committed to backend.
	MigratedKeys(ctx context.Context, backend string) (map[string]struct{}, error)

	// MarkMigrated records that key (of the given size) was copied to backend.
	// It is idempotent — marking an already-recorded key is a no-op.
	MarkMigrated(ctx context.Context, backend, key string, size int64) error
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

// List implements domain.FileBackend.
func (p *PlaceholderBackend) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, fmt.Errorf("placeholder backend (%s) cannot be used directly", p.Title)
}

// Get implements domain.FileBackend.
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
