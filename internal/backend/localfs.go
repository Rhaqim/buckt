package backend

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type LocalFileSystemService struct {
	*logger.BucktLogger
	MediaDir string
	g        singleflight.Group
	cache    domain.LRUCache
}

func NewLocalFileSystemService(bucktLogger *logger.BucktLogger, mediaDir string, cache domain.LRUCache) domain.FileBackend {
	bucktLogger.Info("🚀 Initialising local file system backend")
	return &LocalFileSystemService{
		BucktLogger: bucktLogger,
		MediaDir:    mediaDir,
		g:           singleflight.Group{},
		cache:       cache,
	}
}

func (bfs *LocalFileSystemService) resolve(path string) string {
	return filepath.Join(bfs.MediaDir, path)
}

// Put writes/overwrites a file.
func (bfs *LocalFileSystemService) Put(path string, data []byte) error {
	filePath := bfs.resolve(path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return bfs.WrapError("failed to create directory", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

// Get reads the entire file into memory.
func (bfs *LocalFileSystemService) Get(path string) ([]byte, error) {
	filePath := bfs.resolve(path)

	// check cache first
	if data, ok := bfs.cache.Get(filePath); ok {
		return data, nil
	}

	result, err, _ := bfs.g.Do(filePath, func() (any, error) {
		return os.ReadFile(filePath)
	})
	if err != nil {
		return nil, bfs.WrapError("failed to read file", err)
	}

	bytes := result.([]byte)
	bfs.cache.Add(filePath, bytes)
	return bytes, nil
}

// Stream returns a file stream (caller must Close).
func (bfs *LocalFileSystemService) Stream(path string) (io.ReadCloser, error) {
	filePath := bfs.resolve(path)
	return os.Open(filePath)
}

// Delete removes a file.
func (bfs *LocalFileSystemService) Delete(path string) error {
	filePath := bfs.resolve(path)
	if err := os.Remove(filePath); err != nil {
		return bfs.WrapError("failed to delete file", err)
	}
	return nil
}

// Exists checks if a file exists.
func (bfs *LocalFileSystemService) Exists(path string) (bool, error) {
	filePath := bfs.resolve(path)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, bfs.WrapError("failed to stat file", err)
}

// Stat returns metadata.
func (bfs *LocalFileSystemService) Stat(path string) (*model.FileInfo, error) {
	filePath := bfs.resolve(path)
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, bfs.WrapError("failed to stat file", err)
	}
	return &model.FileInfo{
		Size:         info.Size(),
		LastModified: info.ModTime(),
		ETag:         "",                         // Not applicable locally
		ContentType:  "application/octet-stream", // crude default
	}, nil
}

func (bfs *LocalFileSystemService) DeleteFolder(path string) error {
	dirPath := bfs.resolve(path)
	return os.RemoveAll(dirPath)
}

func (bfs *LocalFileSystemService) Move(oldPath, newPath string) error {
	oldFilePath := filepath.Join(bfs.MediaDir, oldPath)
	newFilePath := filepath.Join(bfs.MediaDir, newPath)

	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		return bfs.WrapError("failed to create directory", err)
	}

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return bfs.WrapError("failed to move file", err)
	}
	return nil
}
