package mocks

import (
	"io"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockFileSystemService struct {
	mock.Mock
}

func NewMockFileSystemService() domain.FileSystemService {
	return &MockFileSystemService{}
}

// FSUpdateFile implements domain.FileSystemService.
func (m *MockFileSystemService) FSUpdateFile(oldPath string, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

// FSValidatePath implements domain.FileSystemService.
func (m *MockFileSystemService) FSValidatePath(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockFileSystemService) FSWriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *MockFileSystemService) FSGetFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystemService) FSGetFileStream(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockFileSystemService) FSDeleteFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// FSDeleteFolder implements domain.FileSystemService.
func (m *MockFileSystemService) FSDeleteFolder(folderPath string) error {
	args := m.Called(folderPath)
	return args.Error(0)
}
