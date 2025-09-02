package domain

import (
	"io"

	"github.com/Rhaqim/buckt/internal/model"
)

type FileBackend interface {
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
	Stat(path string) (*model.FileInfo, error)

	// DeleteFolder removes all objects with the given prefix.
	// For local backend, this will simply remove the directory.
	DeleteFolder(prefix string) error

	Move(oldPath, newPath string) error
}
