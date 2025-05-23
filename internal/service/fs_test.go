package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/stretchr/testify/assert"
)

type MockLRUCache struct{}

func (m *MockLRUCache) Close()                            {}
func (M *MockLRUCache) Get(key string) ([]byte, bool)     { return nil, false }
func (M *MockLRUCache) Add(key string, value []byte) bool { return true }
func (M *MockLRUCache) Hits() uint64                      { return 0 }
func (M *MockLRUCache) Misses() uint64                    { return 0 }

func setupFSTest() (*FileSystemService, string) {
	log := logger.NewLogger("", true, false)
	mediaDir := os.TempDir()
	cache := &MockLRUCache{}
	bfs := NewFileSystemService(log, mediaDir, cache).(*FileSystemService)
	return bfs, mediaDir
}

func TestFSValidatePath(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	testPath := "testfile.txt"
	expectedPath := filepath.Join(mediaDir, testPath)

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)
	defer os.Remove(expectedPath)

	// Validate path
	validatedPath, err := bfs.FSValidatePath(testPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedPath, validatedPath)
}

func TestFSWriteFile(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	testPath := "testfile.txt"
	testContent := []byte("Hello, World!")
	expectedPath := filepath.Join(mediaDir, testPath)

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Write file
	err = bfs.FSWriteFile(testPath, testContent)
	assert.NoError(t, err)
	defer os.Remove(expectedPath)

	// Validate file content
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSGetFile(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	testPath := "testfile.txt"
	testContent := []byte("Hello, World!")
	expectedPath := filepath.Join(mediaDir, testPath)

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Create a test file
	err = os.WriteFile(expectedPath, testContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(expectedPath)

	// Get file
	content, err := bfs.FSGetFile(testPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSUpdateFile(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	oldPath := "oldfile.txt"
	newPath := "newfile.txt"
	testContent := []byte("Hello, World!")
	oldFilePath := filepath.Join(mediaDir, oldPath)
	newFilePath := filepath.Join(mediaDir, newPath)

	// Create a test file
	_, err := os.Create(oldFilePath)
	assert.NoError(t, err)

	// Create a test file
	_, err = os.Create(newFilePath)
	assert.NoError(t, err)

	// Create a test file
	err = os.WriteFile(oldFilePath, testContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(oldFilePath)
	defer os.Remove(newFilePath)

	// Update file
	err = bfs.FSUpdateFile(oldPath, newPath)
	assert.NoError(t, err)

	// Validate old file does not exist
	_, err = os.Stat(oldFilePath)
	assert.True(t, os.IsNotExist(err))

	// Validate new file content
	content, err := os.ReadFile(newFilePath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSDeleteFile(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	testPath := "testfile.txt"
	expectedPath := filepath.Join(mediaDir, testPath)

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Delete file
	err = bfs.FSDeleteFile(testPath)
	assert.NoError(t, err)

	// Validate file does not exist
	_, err = os.Stat(expectedPath)
	assert.True(t, os.IsNotExist(err))
}
func TestFSDeleteFolder(t *testing.T) {
	bfs, mediaDir := setupFSTest()
	testFolderPath := "testfolder"
	expectedFolderPath := filepath.Join(mediaDir, testFolderPath)

	// Create a test folder
	err := os.MkdirAll(expectedFolderPath, os.ModePerm)
	assert.NoError(t, err)

	// Create a test file inside the folder
	testFilePath := filepath.Join(expectedFolderPath, "testfile.txt")
	_, err = os.Create(testFilePath)
	assert.NoError(t, err)

	// Delete folder
	err = bfs.FSDeleteFolder(testFolderPath)
	assert.NoError(t, err)

	// Validate folder does not exist
	_, err = os.Stat(expectedFolderPath)
	assert.True(t, os.IsNotExist(err))
}
