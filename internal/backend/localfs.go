package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"golang.org/x/sync/singleflight"
)

type LocalFileSystemService struct {
	logger   domain.BucktLogger
	mediaDir string
	g        singleflight.Group
	cache    domain.LRUCache
}

func NewLocalFileSystemService(logger domain.BucktLogger, mediaDir string, cache domain.LRUCache) domain.FileBackend {
	logger.Info("🚀 Initialising local file system backend")
	return &LocalFileSystemService{
		logger:   logger,
		mediaDir: mediaDir,
		g:        singleflight.Group{},
		cache:    cache,
	}
}

func (bfs *LocalFileSystemService) Name() string {
	return "local"
}

func (bfs *LocalFileSystemService) resolve(path string) string {
	return filepath.Join(bfs.mediaDir, path)
}

// Put writes/overwrites a file.
func (bfs *LocalFileSystemService) Put(ctx context.Context, path string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return bfs.logger.WrapError("failed to create directory", err)
	}

	tmpPath := filePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return bfs.logger.WrapError("failed to create temp file", err)
	}

	if _, err = f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return bfs.logger.WrapError("failed to write data", err)
	}

	if err = f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return bfs.logger.WrapError("failed to fsync temp file", err)
	}

	if err = f.Close(); err != nil {
		os.Remove(tmpPath)
		return bfs.logger.WrapError("failed to close temp file", err)
	}

	if err = os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return bfs.logger.WrapError("failed to rename temp file", err)
	}

	return nil
}

// Get reads the entire file into memory.
func (bfs *LocalFileSystemService) Get(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)

	// check cache first
	if data, ok := bfs.cache.Get(filePath); ok {
		return data, nil
	}

	result, err, _ := bfs.g.Do(filePath, func() (any, error) {
		return os.ReadFile(filePath)
	})
	if err != nil {
		return nil, bfs.logger.WrapError("failed to read file", err)
	}

	bytes, ok := result.([]byte)
	if !ok {
		return nil, bfs.logger.WrapError("unexpected cache type", fmt.Errorf("expected []byte"))
	}

	bfs.cache.Add(filePath, bytes)
	return bytes, nil
}

// Stream returns a file stream (caller must Close).
func (bfs *LocalFileSystemService) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)
	return os.Open(filePath)
}

// Delete removes a file.
func (bfs *LocalFileSystemService) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)
	if err := os.Remove(filePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return bfs.logger.WrapError("failed to delete file", err)
	}

	return nil
}

// Exists checks if a file exists.
func (bfs *LocalFileSystemService) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, bfs.logger.WrapError("failed to stat file", err)
}

// Stat returns metadata.
func (bfs *LocalFileSystemService) Stat(ctx context.Context, path string) (*model.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	filePath := bfs.resolve(path)
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, bfs.logger.WrapError("failed to stat file", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &model.FileInfo{
		Size:         info.Size(),
		LastModified: info.ModTime(),
		ETag:         "", // Not applicable locally
		ContentType:  contentType,
	}, nil
}

func (bfs *LocalFileSystemService) DeleteFolder(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dirPath := bfs.resolve(path)
	return os.RemoveAll(dirPath)
}

func (bfs *LocalFileSystemService) Move(ctx context.Context, oldPath, newPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	oldFilePath := filepath.Join(bfs.mediaDir, oldPath)
	newFilePath := filepath.Join(bfs.mediaDir, newPath)

	if _, err := os.Stat(oldFilePath); errors.Is(err, fs.ErrNotExist) {
		return bfs.logger.WrapError("source file does not exist", err)
	}

	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		return bfs.logger.WrapError("failed to create directory", err)
	}

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err == syscall.EXDEV {
			// cross-device move fallback: copy then delete
			if copyErr := copyFile(oldFilePath, newFilePath); copyErr != nil {
				return bfs.logger.WrapError("cross-device copy failed", copyErr)
			}
			if delErr := os.Remove(oldFilePath); delErr != nil {
				return bfs.logger.WrapError("failed to remove old file after cross-device move", delErr)
			}
			return nil
		}
		return bfs.logger.WrapError("failed to move file", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		os.Remove(dst)
		return err
	}
	if err = out.Sync(); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}
