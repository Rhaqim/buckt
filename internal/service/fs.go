package service

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type FileSystemService struct {
	*logger.BucktLogger

	MediaDir string

	g     singleflight.Group
	cache domain.LRUCache
}

func NewFileSystemService(bucktLogger *logger.BucktLogger, medaiDir string, cache domain.LRUCache) domain.FileSystemService {
	bucktLogger.Info("🚀 Initialising file system services")
	return &FileSystemService{
		BucktLogger: bucktLogger,
		MediaDir:    medaiDir,

		g:     singleflight.Group{},
		cache: cache,
	}
}

func (bfs *FileSystemService) FSValidatePath(path string) (string, error) {
	filePath := filepath.Join(bfs.MediaDir, path)

	if _, err := os.Stat(filePath); err != nil {
		return "", bfs.WrapError("failed to validate file path", err)
	}

	return filePath, nil
}

func (bfs *FileSystemService) FSWriteFile(filePath string, file []byte) error {
	// File system path
	filePath = filepath.Join(bfs.MediaDir, filePath)

	// Save the file to the file system
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return bfs.WrapError("failed to create directory", err)
	}

	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return bfs.WrapError("failed to write file", err)
	}

	return nil
}

func (bfs *FileSystemService) FSGetFile(path string) ([]byte, error) {
	filePath, err := bfs.FSValidatePath(path)
	if err != nil {
		return nil, err
	}

	if file, ok := bfs.cache.Get(filePath); ok {
		return file, nil
	}

	result, err, _ := bfs.g.Do(filePath, func() (any, error) {
		return os.ReadFile(filePath)
	})

	if err != nil {
		return nil, bfs.WrapError("failed to read file", err)
	}

	// check what type result is
	if _, ok := result.([]byte); !ok {
		return nil, bfs.WrapError("failed to read file, type mismatch", err)
	}
	// bfs.cache.Add(filePath, result.([]byte))
	bfs.cache.Add(filePath, result.([]byte))

	return result.([]byte), nil
}

func (bfs *FileSystemService) FSGetFileStream(path string) (io.ReadCloser, error) {
	filePath, err := bfs.FSValidatePath(path)
	if err != nil {
		return nil, err
	}

	// if file, ok := bfs.cache.Get(filePath); ok {
	// 	return file.(*os.File), nil
	// }

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	// bfs.cache.Add(filePath, file)

	return file, nil // Caller should close the file after reading
}

func (bfs *FileSystemService) FSUpdateFile(oldPath, newPath string) error {
	oldFilePath, err := bfs.FSValidatePath(oldPath)
	if err != nil {
		return err
	}

	newFilePath, err := bfs.FSValidatePath(newPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		return bfs.WrapError("failed to create directory", err)
	}

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return bfs.WrapError("failed to update file", err)
	}

	return nil
}

func (bfs *FileSystemService) FSDeleteFile(folderPath string) error {
	filePath, err := bfs.FSValidatePath(folderPath)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return bfs.WrapError("failed to delete file", err)
	}

	return nil
}

func (bfs *FileSystemService) FSDeleteFolder(folderPath string) error {
	folderPath = filepath.Join(bfs.MediaDir, folderPath)

	if err := os.RemoveAll(folderPath); err != nil {
		return bfs.WrapError("failed to delete folder", err)
	}

	return nil
}
