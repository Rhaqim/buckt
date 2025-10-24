package mocks

import (
	"context"
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/stretchr/testify/mock"
)

type LocalFileSystemService struct {
	mock.Mock
}

var _ domain.FileBackend = (*LocalFileSystemService)(nil)

func (m *LocalFileSystemService) Name() string {
	return "LocalFileSystem"
}

// FSUpdateFile implements domain.FileSystemService.
func (m *LocalFileSystemService) Move(ctx context.Context, oldPath string, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

func (m *LocalFileSystemService) Put(ctx context.Context, path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *LocalFileSystemService) Get(ctx context.Context, path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *LocalFileSystemService) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *LocalFileSystemService) Delete(ctx context.Context, path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// FSDeleteFolder implements domain.FileSystemService.
func (m *LocalFileSystemService) DeleteFolder(ctx context.Context, folderPath string) error {
	args := m.Called(folderPath)
	return args.Error(0)
}

// Exists implements domain.FileBackend.
func (m *LocalFileSystemService) Exists(ctx context.Context, path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

// Stat implements domain.FileBackend.
func (m *LocalFileSystemService) Stat(ctx context.Context, path string) (*model.FileInfo, error) {
	args := m.Called(path)
	return args.Get(0).(*model.FileInfo), args.Error(1)
}
