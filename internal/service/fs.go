package service

import (
	"os"
	"path/filepath"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/logger"
)

type FileSystemService struct {
	*logger.Logger

	MediaDir string
}

func NewFileSystemService(log *logger.Logger, medaiDir string) domain.FileSystemService {
	return &FileSystemService{
		Logger:   log,
		MediaDir: medaiDir,
	}
}

func (bfs *FileSystemService) FSValidatePath(path string) (string, error) {
	filePath := filepath.Join(bfs.MediaDir, path)

	if _, err := os.Stat(filePath); err != nil {
		return "", bfs.WrapError("failed to validate file path", err)
	}

	return filePath, nil
}

func (bfs *FileSystemService) FSWriteFile(path string, file []byte) error {
	// File system path
	filePath, err := bfs.FSValidatePath(path)
	if err != nil {
		return err
	}

	// Save the file to the file system
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
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

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, bfs.WrapError("failed to read file", err)
	}

	return file, nil
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
