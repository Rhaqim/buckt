package service_old

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/domain_old"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type BucktFSService struct {
	*logger.Logger
	*config.Config
}

func NewBucktFSService(log *logger.Logger, cfg *config.Config) domain_old.BucktFileSystemService {

	return &BucktFSService{log, cfg}
}

func (bfs *BucktFSService) FSValidatePath(path string) (string, error) {
	filePath := filepath.Join(bfs.MediaDir, path)

	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	return filePath, nil
}

func (bfs *BucktFSService) FSWriteFile(path string, file []byte) error {
	// File system path
	filePath := filepath.Join(bfs.MediaDir, path)

	// Save the file to the file system
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (bfs *BucktFSService) FSGetFile(path string) ([]byte, error) {
	filePath, err := bfs.FSValidatePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to validate file path: %w", err)
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return file, nil
}

func (bfs *BucktFSService) FSUpdateFile(oldPath, newPath string) error {
	oldFilePath, err := bfs.FSValidatePath(oldPath)
	if err != nil {
		return fmt.Errorf("failed to validate old file path: %w", err)
	}

	newFilePath, err := bfs.FSValidatePath(newPath)
	if err != nil {
		return fmt.Errorf("failed to validate new file path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(newFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func (bfs *BucktFSService) FSDeleteFile(folderPath string) error {
	filePath, err := bfs.FSValidatePath(folderPath)
	if err != nil {
		return fmt.Errorf("failed to validate file path: %w", err)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
