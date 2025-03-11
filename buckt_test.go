package buckt

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func UUIDFromString(name string) uuid.UUID {
	namespace := uuid.NameSpaceDNS // You can change this to another namespace if needed
	return uuid.NewSHA1(namespace, []byte(name))
}

type MockBuckt struct {
	*Buckt
	*mocks.MockFileService
	*mocks.MockFolderService
}

func setupBucktTest(t *testing.T) MockBuckt {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer sqlDB.Close()

	bucktOpts := BucktConfig{
		DB:             DBConfig{Driver: SQLite, Database: sqlDB},
		Log:            LogConfig{LogTerminal: false, Debug: false},
		MediaDir:       "media",
		StandaloneMode: true,
		FlatNameSpaces: false,
	}

	buckt, err := New(bucktOpts)
	assert.NoError(t, err)
	assert.NotNil(t, buckt)

	mockFileService := new(mocks.MockFileService)
	mockFolderService := new(mocks.MockFolderService)

	buckt.fileService = mockFileService
	buckt.folderService = mockFolderService

	return MockBuckt{
		Buckt:             buckt,
		MockFileService:   mockFileService,
		MockFolderService: mockFolderService,
	}
}

func TestNew(t *testing.T) {
	t.Run("SQLite with provided instance", func(t *testing.T) {
		mockBuckt := setupBucktTest(t)

		assert.NotNil(t, mockBuckt)
	})

	t.Run("SQLite without provided instance", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		bucktOpts := BucktConfig{
			DB:             DBConfig{Driver: SQLite, Database: sqlDB},
			Log:            LogConfig{LogTerminal: false, Debug: false},
			MediaDir:       "media",
			StandaloneMode: true,
			FlatNameSpaces: false,
		}

		buckt, err := New(bucktOpts)
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	// t.Run("Postgres with provided instance", func(t *testing.T) {
	// 	sqlDB, err := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	// 	assert.NoError(t, err)
	// 	defer sqlDB.Close()

	// 	bucktOpts := BucktConfig{
	// 		DB:             DBConfig{Driver: Postgres, Database: sqlDB},
	// 		Log:            LogConfig{LogTerminal: false, Debug: false},
	// 		MediaDir:       "media",
	// 		StandaloneMode: true,
	// 		FlatNameSpaces: false,
	// 	}

	// 	buckt, err := New(bucktOpts)
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, buckt)
	// })

	t.Run("Postgres without provided instance", func(t *testing.T) {
		bucktOpts := BucktConfig{
			DB:             DBConfig{Driver: Postgres, Database: nil},
			Log:            LogConfig{LogTerminal: false, Debug: false},
			MediaDir:       "media",
			StandaloneMode: true,
			FlatNameSpaces: false,
		}

		buckt, err := New(bucktOpts)
		assert.Error(t, err)
		assert.Nil(t, buckt)
	})

}

func TestDefault(t *testing.T) {
	t.Run("With Default", func(t *testing.T) {
		buckt, err := Default()
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Standalone", func(t *testing.T) {
		buckt, err := Default(StandaloneMode(true))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With FlatNameSpaces", func(t *testing.T) {
		buckt, err := Default(FlatNameSpaces(true))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With MediaDir", func(t *testing.T) {
		buckt, err := Default(MediaDir("media"))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Cache", func(t *testing.T) {
		buckt, err := Default(WithCache(cache.NewNoOpCache()))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Log", func(t *testing.T) {
		buckt, err := Default(WithLog(LogConfig{LogTerminal: false, Debug: false}))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With DB", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		buckt, err := Default(WithDB(SQLite, sqlDB))
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With all options", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		buckt, err := Default(
			StandaloneMode(true),
			FlatNameSpaces(true),
			MediaDir("media"),
			WithCache(cache.NewNoOpCache()),
			WithLog(LogConfig{LogTerminal: false, Debug: false}),
			WithDB(SQLite, sqlDB),
		)
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})
}

func TestGetHandler(t *testing.T) {
	buckt := setupBucktTest(t)

	handler := buckt.GetHandler()
	assert.NotNil(t, handler)
}

func TestServer(t *testing.T) {
	buckt := setupBucktTest(t)

	go func() {
		err := buckt.StartServer(":8080")
		assert.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond) // Ensure server starts before test exits

	// Cleanup to ensure the server is closed after the test
	t.Cleanup(func() {
		buckt.Close()
	})
}

func TestClose(t *testing.T) {
	buckt := setupBucktTest(t)
	assert.NotNil(t, buckt)

	buckt.Close()
}

func TestNewFolder(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Expected folder ID
	expectedFolderID := "550e8400-e29b-41d4-a716-446655440000"

	// Mocking the CreateFolder method
	buckt.MockFolderService.On("CreateFolder", "user1", "550e8400-e29b-41d4-a716-446655440001", "folder1", "description1").
		Return(expectedFolderID, nil)

	// Call the method
	actualFolderID, err := buckt.NewFolder("user1", "550e8400-e29b-41d4-a716-446655440001", "folder1", "description1")

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedFolderID, actualFolderID) // Compare the expected and actual values

	// Verify mock expectations
	buckt.MockFolderService.AssertExpectations(t)
}

func TestListFolders(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	expectedFolders := []model.FolderModel{
		{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), Name: "folder1", Description: "description1"},
		{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), Name: "folder2", Description: "description2"},
	}

	buckt.MockFolderService.On("GetFolders", "550e8400-e29b-41d4-a716-446655440002").
		Return(expectedFolders, nil)

	// Call the method
	folders, err := buckt.ListFolders("550e8400-e29b-41d4-a716-446655440002")
	assert.NoError(t, err)
	assert.Equal(t, expectedFolders, folders)

	// Verify expectations
	buckt.MockFolderService.AssertExpectations(t)
}

func TestGetFolderWithContent(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	expectedFolder := model.FolderModel{
		ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:        "folder1",
		Description: "description1",
		Files: []model.FileModel{
			{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), Name: "file1", ContentType: "text/plain", Data: []byte("file content")},
			{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"), Name: "file2", ContentType: "text/plain", Data: []byte("file content")},
		},
		Folders: []model.FolderModel{
			{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440003"), Name: "folder2", Description: "description2"},
			{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440004"), Name: "folder3", Description: "description3"},
		},
	}

	buckt.MockFolderService.On("GetFolder", "user1", "550e8400-e29b-41d4-a716-446655440000").
		Return(&expectedFolder, nil)

	// Call the method
	folder, err := buckt.GetFolderWithContent("user1", "550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.NotNil(t, folder)

	// Verify expectations
	buckt.MockFolderService.AssertExpectations(t)
}

func TestMoveFolder(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFolderService.On("MoveFolder", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001").
		Return(nil)

	// Call the method
	err := buckt.Buckt.MoveFolder("user1", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFolderService.AssertExpectations(t)

}

func TestDeleteFolder(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFolderService.On("DeleteFolder", "550e8400-e29b-41d4-a716-446655440000").
		Return("550e8400-e29b-41d4-a716-446655440001", nil)

	// Call the method
	err := buckt.Buckt.DeleteFolder("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFolderService.AssertExpectations(t)
}

func TestDeleteFolderPermanently(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFolderService.On("ScrubFolder", "user1", "550e8400-e29b-41d4-a716-446655440000").
		Return("parent1", nil)

	// Call the method
	err := buckt.DeleteFolderPermanently("user1", "550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFolderService.AssertExpectations(t)
}

func TestUploadFile(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFileService.On("CreateFile", "user1", "folder1", "file1", "text/plain", []byte("file content")).
		Return("550e8400-e29b-41d4-a716-446655440000", nil)

	// Call the method
	fileID, err := buckt.UploadFile("user1", "folder1", "file1", "text/plain", []byte("file content"))
	assert.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", fileID)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestGetFile(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	expectedFile := model.FileModel{
		ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:        "file1",
		ContentType: "text/plain",
		Data:        []byte("file content"),
	}
	buckt.MockFileService.On("GetFile", "550e8400-e29b-41d4-a716-446655440000").
		Return(&expectedFile, nil)

	// Call the method
	file, err := buckt.Buckt.GetFile("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.NotNil(t, file)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestListFiles(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	expectedFiles := []model.FileModel{
		{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), Name: "file1", ContentType: "text/plain", Data: []byte("file content")},
		{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), Name: "file2", ContentType: "text/plain", Data: []byte("file content")},
	}

	buckt.MockFileService.On("GetFiles", "550e8400-e29b-41d4-a716-446655440002").
		Return(expectedFiles, nil)

	// Call the method
	files, err := buckt.ListFiles("550e8400-e29b-41d4-a716-446655440002")
	assert.NoError(t, err)
	assert.Equal(t, expectedFiles, files)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestMoveFile(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFileService.On("MoveFile", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001").
		Return(nil)

	// Call the method
	err := buckt.Buckt.MoveFile("550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestDeleteFile(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFileService.On("DeleteFile", "550e8400-e29b-41d4-a716-446655440000").
		Return("parent1", nil)

	// Call the method
	err := buckt.Buckt.DeleteFile("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestDeleteFilePermanently(t *testing.T) {
	buckt := setupBucktTest(t)

	// Ensure cleanup after test execution
	t.Cleanup(func() {
		buckt.Close() // Assuming there's a method to clean up resources
	})

	// Mock the expected behavior
	buckt.MockFileService.On("ScrubFile", "550e8400-e29b-41d4-a716-446655440000").
		Return("parent1", nil)

	// Call the method
	err := buckt.DeleteFilePermanently("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}
