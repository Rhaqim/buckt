package backend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func setupFSV2Test() (*LocalFileSystemService, string) {
	log := logger.NewLogger("", true, false)
	mediaDir := os.TempDir()
	cache := new(mocks.NoopLRUCache)
	bfs := NewLocalFileSystemService(log, mediaDir, cache).(*LocalFileSystemService)
	return bfs, mediaDir
}

func TestFSV2Put(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	testPath := "testfile.txt"
	testContent := []byte("Hello, World!")
	expectedPath := filepath.Join(mediaDir, testPath)
	ctx := t.Context()

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Write file
	err = bfs.Put(ctx, testPath, testContent)
	assert.NoError(t, err)
	defer os.Remove(expectedPath)

	// Validate file content
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSV2GetFile(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	testPath := "testfile.txt"
	testContent := []byte("Hello, World!")
	expectedPath := filepath.Join(mediaDir, testPath)
	ctx := t.Context()

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Create a test file
	err = os.WriteFile(expectedPath, testContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(expectedPath)

	// Get file
	content, err := bfs.Get(ctx, testPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSV2GetNonExistentFile(t *testing.T) {
	bfs, _ := setupFSV2Test()
	nonExistentPath := "nonexistentfile.txt"
	ctx := t.Context()

	// Get non-existent file
	_, err := bfs.Get(ctx, nonExistentPath)
	assert.Error(t, err)
}

func TestFSV2List(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	testFolderPath := "testfolder"
	expectedFolderPath := filepath.Join(mediaDir, testFolderPath)
	ctx := t.Context()

	// Create a test folder
	err := os.MkdirAll(expectedFolderPath, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(expectedFolderPath)

	// Create test files inside the folder
	testFile1Path := filepath.Join(expectedFolderPath, "file1.txt")
	_, err = os.Create(testFile1Path)
	assert.NoError(t, err)

	testFile2Path := filepath.Join(expectedFolderPath, "file2.txt")
	_, err = os.Create(testFile2Path)
	assert.NoError(t, err)

	// List files
	files, err := bfs.List(ctx, testFolderPath)
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "testfolder/file1.txt")
	assert.Contains(t, files, "testfolder/file2.txt")
}

func TestFSV2Move(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	oldPath := "oldfile.txt"
	newPath := "newfile.txt"
	testContent := []byte("Hello, World!")
	oldFilePath := filepath.Join(mediaDir, oldPath)
	newFilePath := filepath.Join(mediaDir, newPath)
	ctx := t.Context()

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

	// Move file
	err = bfs.Move(ctx, oldPath, newPath)
	assert.NoError(t, err)

	// Validate old file does not exist
	_, err = os.Stat(oldFilePath)
	assert.True(t, os.IsNotExist(err))

	// Validate new file content
	content, err := os.ReadFile(newFilePath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestFSV2DeleteFile(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	testPath := "testfile.txt"
	expectedPath := filepath.Join(mediaDir, testPath)
	ctx := t.Context()

	// Create a test file
	_, err := os.Create(expectedPath)
	assert.NoError(t, err)

	// Delete file
	err = bfs.Delete(ctx, testPath)
	assert.NoError(t, err)

	// Validate file does not exist
	_, err = os.Stat(expectedPath)
	assert.True(t, os.IsNotExist(err))
}
func TestFSV2DeleteFolder(t *testing.T) {
	bfs, mediaDir := setupFSV2Test()
	testFolderPath := "testfolder"
	expectedFolderPath := filepath.Join(mediaDir, testFolderPath)
	ctx := t.Context()

	// Create a test folder
	err := os.MkdirAll(expectedFolderPath, os.ModePerm)
	assert.NoError(t, err)

	// Create a test file inside the folder
	testFilePath := filepath.Join(expectedFolderPath, "testfile.txt")
	_, err = os.Create(testFilePath)
	assert.NoError(t, err)

	// Delete folder
	err = bfs.DeleteFolder(ctx, testFolderPath)
	assert.NoError(t, err)

	// Validate folder does not exist
	_, err = os.Stat(expectedFolderPath)
	assert.True(t, os.IsNotExist(err))
}
