package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type BucktFSService struct {
	*logger.Logger
	*config.Config
}

func NewBucktFSService(log *logger.Logger, cfg *config.Config) domain.BucktFileSystemService {

	return &BucktFSService{log, cfg}
}

func (bfs *BucktFSService) FSGetFile(path string) ([]byte, error) {
	filePath := filepath.Join(bfs.Media.Dir, path)

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return file, nil
}

func (bfs *BucktFSService) FSWriteFile(path string, file []byte) error {
	// File system path
	filePath := filepath.Join(bfs.Media.Dir, path)

	// Save the file to the file system
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (bfs *BucktFSService) FSUpdateFile(oldPath, newPath string) error {
	oldFilePath := filepath.Join(bfs.Media.Dir, oldPath)
	newFilePath := filepath.Join(bfs.Media.Dir, newPath)

	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func (bfs *BucktFSService) FSDeleteFile(folderPath string) error {
	filePath := filepath.Join(bfs.Media.Dir, folderPath)

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (bfs *BucktFSService) FSValidatePath(path string) (string, error) {
	filePath := filepath.Join(bfs.Media.Dir, path)

	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	return filePath, nil
}
