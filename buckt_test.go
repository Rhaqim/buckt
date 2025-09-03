package buckt

import (
	"bytes"
	"database/sql"
	"io"
	"testing"

	"github.com/Rhaqim/buckt/internal/backend"
	"github.com/Rhaqim/buckt/internal/database"
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
	*Client
	MockFileService   *mocks.FileService
	MockFolderService *mocks.FolderService
}

func setup(t *testing.T, bucktOpts Config) MockBuckt {
	buckt, err := New(bucktOpts)
	assert.NoError(t, err)
	assert.NotNil(t, buckt)

	mockFileService := new(mocks.FileService)
	mockFolderService := new(mocks.FolderService)

	buckt.fileService = mockFileService
	buckt.folderService = mockFolderService

	return MockBuckt{
		Client:            buckt,
		MockFileService:   mockFileService,
		MockFolderService: mockFolderService,
	}
}

func setupBucktTest(t *testing.T) MockBuckt {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer sqlDB.Close()

	bucktOpts := Config{
		DB:             DBConfig{Driver: SQLite, Database: sqlDB},
		Log:            LogConfig{LogTerminal: false, Debug: false},
		MediaDir:       "media",
		FlatNameSpaces: false,
	}

	return setup(t, bucktOpts)
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

		bucktOpts := Config{
			DB:             DBConfig{Driver: SQLite, Database: sqlDB},
			Log:            LogConfig{LogTerminal: false, Debug: false},
			MediaDir:       "media",
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

	// 	bucktOpts := Config{
	// 		DB:             DBConfig{Driver: Postgres, Database: sqlDB},
	// 		Log:            LogConfig{LogTerminal: false, Debug: false},
	// 		MediaDir:       "media",
	//
	// 		FlatNameSpaces: false,
	// 	}

	// 	buckt, err := New(bucktOpts)
	// 	// assert.NoError(t, err)
	// 	// assert.NotNil(t, buckt)
	// 	assert.Error(t, err) // 🚨 Error: pq: database "postgres" does not exist
	// 	assert.Nil(t, buckt)
	// })

	t.Run("Postgres without provided instance", func(t *testing.T) {
		bucktOpts := Config{
			DB:       DBConfig{Driver: Postgres, Database: nil},
			Log:      LogConfig{LogTerminal: false, Debug: false},
			MediaDir: "media",

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
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})

		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With FlatNameSpaces", func(t *testing.T) {
		buckt, err := Default(FlatNameSpaces(true))
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With MediaDir", func(t *testing.T) {
		buckt, err := Default(MediaDir("media"))
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Cache", func(t *testing.T) {
		fileCacheConfig := FileCacheConfig{
			NumCounters: 1e7,     // 10M
			MaxCost:     1 << 30, // 1GB
			BufferItems: 64,
		}

		cacheConfig := CacheConfig{
			Manager:         mocks.NewNoopCache(),
			FileCacheConfig: fileCacheConfig,
		}

		buckt, err := Default(WithCache(cacheConfig))
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Log", func(t *testing.T) {
		buckt, err := Default(WithLog(LogConfig{LogTerminal: false, Debug: false}))
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With DB", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		buckt, err := Default(WithDB(SQLite, sqlDB))
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With all options", func(t *testing.T) {
		sqlDB, err := sql.Open("sqlite3", ":memory:")
		assert.NoError(t, err)
		defer sqlDB.Close()

		fileCacheConfig := FileCacheConfig{
			NumCounters: 1e7,     // 10M
			MaxCost:     1 << 30, // 1GB
			BufferItems: 64,
		}

		cacheConfig := CacheConfig{
			Manager:         mocks.NewNoopCache(),
			FileCacheConfig: fileCacheConfig,
		}

		buckt, err := Default(
			FlatNameSpaces(true),
			MediaDir("media"),
			WithCache(cacheConfig),
			WithLog(LogConfig{LogTerminal: false, Debug: false}),
			WithDB(SQLite, sqlDB),
		)
		assert.NoError(t, err)
		assert.NotNil(t, buckt)
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
	err := buckt.Client.MoveFolder("user1", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001")
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
	_, err := buckt.Client.DeleteFolder("550e8400-e29b-41d4-a716-446655440000")
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
	_, err := buckt.DeleteFolderPermanently("user1", "550e8400-e29b-41d4-a716-446655440000")
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
	file, err := buckt.Client.GetFile("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.NotNil(t, file)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestGetFileStream(t *testing.T) {
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
	expectedStream := io.NopCloser(bytes.NewReader([]byte("file content")))

	buckt.MockFileService.On("GetFileStream", "550e8400-e29b-41d4-a716-446655440000").
		Return(&expectedFile, expectedStream, nil)

	// Call the method
	file, stream, err := buckt.Client.GetFileStream("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.NotNil(t, stream)
	defer stream.Close()

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
	err := buckt.Client.MoveFile("550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001")
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
	_, err := buckt.Client.DeleteFile("550e8400-e29b-41d4-a716-446655440000")
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
	_, err := buckt.DeleteFilePermanently("550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)

	// Verify expectations
	buckt.MockFileService.AssertExpectations(t)
}

func TestInitializeCache(t *testing.T) {
	// Mock logger
	mockLogger := &mocks.NoopLogger{}

	t.Run("returns provided cache manager and valid lruCache", func(t *testing.T) {
		fileCacheConfig := FileCacheConfig{
			NumCounters: 1e3,
			MaxCost:     1 << 20,
			BufferItems: 8,
		}
		providedCache := mocks.NewNoopCache()
		conf := CacheConfig{
			Manager:         providedCache,
			FileCacheConfig: fileCacheConfig,
		}

		cm, lru := initializeCache(conf, mockLogger)
		assert.Equal(t, providedCache, cm)
		assert.NotNil(t, lru)
	})

	t.Run("returns NoOpCache if Manager is nil", func(t *testing.T) {
		fileCacheConfig := FileCacheConfig{
			NumCounters: 1e3,
			MaxCost:     1 << 20,
			BufferItems: 8,
		}
		conf := CacheConfig{
			Manager:         nil,
			FileCacheConfig: fileCacheConfig,
		}

		cm, lru := initializeCache(conf, mockLogger)
		assert.NotNil(t, cm)
		assert.NotNil(t, lru)
	})

	t.Run("fallbacks to NoOpFileCache on error", func(t *testing.T) {
		// Use invalid config to force error
		fileCacheConfig := FileCacheConfig{
			NumCounters: 0,
			MaxCost:     0,
			BufferItems: 0,
		}
		conf := CacheConfig{
			Manager:         nil,
			FileCacheConfig: fileCacheConfig,
		}

		cm, lru := initializeCache(conf, mockLogger)
		assert.NotNil(t, cm)
		assert.NotNil(t, lru)
	})
}

func TestResolveBackend(t *testing.T) {
	mockLogger := &mocks.NoopLogger{}
	mockLRU := &mocks.NoopLRUCache{}
	mediaDir := "media"

	t.Run("MigrationEnabled with Source and Target", func(t *testing.T) {
		source := &mocks.Backend{NameVal: "local"}
		target := &mocks.Backend{NameVal: "mock"}
		bc := BackendConfig{
			MigrationEnabled: true,
			Source:           source,
			Target:           target,
		}
		result := resolveBackend(mediaDir, bc, mockLogger, mockLRU)
		_, ok := result.(*backend.MigrationBackendService)
		assert.True(t, ok)
	})

	t.Run("Source only", func(t *testing.T) {
		source := &mocks.Backend{NameVal: "local"}
		bc := BackendConfig{
			Source: source,
		}
		result := resolveBackend(mediaDir, bc, mockLogger, mockLRU)
		// Should instantiate local backend
		_, ok := result.(*backend.LocalFileSystemService)
		assert.True(t, ok)
	})

	t.Run("Target only", func(t *testing.T) {
		target := &mocks.Backend{NameVal: "local"}
		bc := BackendConfig{
			Target: target,
		}
		result := resolveBackend(mediaDir, bc, mockLogger, mockLRU)
		_, ok := result.(*backend.LocalFileSystemService)
		assert.True(t, ok)
	})

	t.Run("No Source or Target", func(t *testing.T) {
		bc := BackendConfig{}
		result := resolveBackend(mediaDir, bc, mockLogger, mockLRU)
		_, ok := result.(*backend.LocalFileSystemService)
		assert.True(t, ok)
	})
}

func TestInstantiateIfLocal(t *testing.T) {
	mockLogger := &mocks.NoopLogger{}
	mockLRU := &mocks.NoopLRUCache{}
	mediaDir := "media"

	t.Run("Returns LocalFileSystemService if backend name is local", func(t *testing.T) {
		b := &mocks.Backend{NameVal: "local"}
		result := instantiateIfLocal(b, mediaDir, mockLogger, mockLRU)
		_, ok := result.(*backend.LocalFileSystemService)
		assert.True(t, ok)
	})

	t.Run("Returns backend as is if name is not local", func(t *testing.T) {
		b := &mocks.Backend{NameVal: "mock"}
		result := instantiateIfLocal(b, mediaDir, mockLogger, mockLRU)
		assert.Equal(t, b, result)
	})
}

func TestNewAppServices(t *testing.T) {
	mockLogger := &mocks.NoopLogger{}
	mockCacheManager := &mocks.NoopCache{}
	mockBackend := &mocks.Backend{}
	// Use an in-memory SQLite DB for testing
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer sqlDB.Close()

	dbConf := DBConfig{Driver: SQLite, Database: sqlDB}
	db, err := database.NewDB(dbConf.Database, dbConf.Driver, mockLogger, false)
	assert.NoError(t, err)
	defer db.Close()

	t.Run("returns valid FolderService and FileService", func(t *testing.T) {
		folderService, fileService := newAppServices(
			true,
			db,
			mockLogger,
			mockCacheManager,
			mockBackend,
		)
		assert.NotNil(t, folderService)
		assert.NotNil(t, fileService)
	})

	t.Run("returns different instances for FolderService and FileService", func(t *testing.T) {
		folderService, fileService := newAppServices(
			false,
			db,
			mockLogger,
			mockCacheManager,
			mockBackend,
		)
		assert.NotNil(t, folderService)
		assert.NotNil(t, fileService)
		assert.NotEqual(t, folderService, fileService)
	})
}
