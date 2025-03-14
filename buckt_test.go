package buckt

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Rhaqim/buckt/internal/cache"
	"github.com/Rhaqim/buckt/internal/domain"
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
	*mocks.MockCloudService
}

func setup(t *testing.T, bucktOpts BucktConfig) MockBuckt {
	buckt, err := New(bucktOpts)
	assert.NoError(t, err)
	assert.NotNil(t, buckt)

	mockFileService := new(mocks.MockFileService)
	mockFolderService := new(mocks.MockFolderService)
	mockCloudService := new(mocks.MockCloudService)

	buckt.fileService = mockFileService
	buckt.folderService = mockFolderService
	buckt.cloudService = mockCloudService

	return MockBuckt{
		Buckt:             buckt,
		MockFileService:   mockFileService,
		MockFolderService: mockFolderService,
		MockCloudService:  mockCloudService,
	}
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

	return setup(t, bucktOpts)
}

func setupCloudTest(t *testing.T, config CloudConfig) MockBuckt {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	// Ensure database is properly closed after the test
	t.Cleanup(func() {
		sqlDB.Close()
	})

	bucktOpts := BucktConfig{
		DB:             DBConfig{Driver: SQLite, Database: sqlDB},
		Log:            LogConfig{LogTerminal: false, Debug: false},
		MediaDir:       "media",
		StandaloneMode: true,
		FlatNameSpaces: false,
		Cloud:          config,
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
	// 	// assert.NoError(t, err)
	// 	// assert.NotNil(t, buckt)
	// 	assert.Error(t, err) // 🚨 Error: pq: database "postgres" does not exist
	// 	assert.Nil(t, buckt)
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
		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})

		assert.NoError(t, err)
		assert.NotNil(t, buckt)
	})

	t.Run("With Standalone", func(t *testing.T) {
		buckt, err := Default(StandaloneMode(true))
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
		buckt, err := Default(WithCache(cache.NewNoOpCache()))
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

	t.Run("With Cloud", func(t *testing.T) {
		cloudConfig := CloudConfig{
			Provider: CloudProviderAWS,
			Credentials: AWSConfig{
				AccessKey: "accessKey",
				SecretKey: "secretKey",
				Region:    "region",
				Bucket:    "bucket",
			},
		}
		buckt, err := Default(WithCloud(cloudConfig))
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

func TestTransferFile(t *testing.T) {
	t.Run("With CloudProviderNone", func(t *testing.T) {
		config := CloudConfig{Provider: CloudProviderNone}

		buckt := setupBucktTest(t)

		// set cloudService to nil to simulate an error
		buckt.cloudService = nil

		err := buckt.InitCloudService(config)
		assert.Error(t, err)
		assert.Nil(t, buckt.cloudService)

		buckt.MockCloudService.On("UploadFileToCloud", "550e8400-e29b-41d4-a716-446655440000").
			Return(nil)

		// Call the method
		err = buckt.Buckt.TransferFile("550e8400-e29b-41d4-a716-446655440000")
		assert.Error(t, err)

		// Verify expectations
		buckt.MockFileService.AssertExpectations(t)
	})

	t.Run("With CloudProviderAWS", func(t *testing.T) {
		config := CloudConfig{
			Provider: CloudProviderAWS,
			Credentials: AWSConfig{
				AccessKey: "accessKey",
				SecretKey: "secretKey",
				Region:    "region",
				Bucket:    "bucket",
			},
		}

		buckt := setupBucktTest(t)

		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})

		// set cloudService to nil to simulate an error
		buckt.cloudService = nil

		err := buckt.InitCloudService(config)
		assert.NoError(t, err)
		assert.NotNil(t, buckt.cloudService)

		// Add mock cloud service]
		buckt.cloudService = buckt.MockCloudService

		// Mock the expected behavior
		buckt.MockCloudService.On("UploadFileToCloud", "550e8400-e29b-41d4-a716-446655440000").
			Return(nil)

		// Call the method
		err = buckt.Buckt.TransferFile("550e8400-e29b-41d4-a716-446655440000")
		assert.NoError(t, err)

		// Verify expectations
		buckt.MockFileService.AssertExpectations(t)
	})
}

func TestTransferFolder(t *testing.T) {
	t.Run("With CloudProviderNone", func(t *testing.T) {
		config := CloudConfig{Provider: CloudProviderNone}

		buckt := setupBucktTest(t)

		// set cloudService to nil to simulate an error
		buckt.cloudService = nil

		err := buckt.InitCloudService(config)
		assert.Error(t, err)
		assert.Nil(t, buckt.cloudService)

		buckt.MockCloudService.On("UploadFolderToCloud", "user_1", "550e8400-e29b-41d4-a716-446655440000").
			Return(nil)

		// Call the method
		err = buckt.Buckt.TransferFolder("user_1", "550e8400-e29b-41d4-a716-446655440000")
		assert.Error(t, err)

		// Verify expectations
		buckt.MockFolderService.AssertExpectations(t)
	})

	t.Run("With CloudProviderAWS", func(t *testing.T) {
		config := CloudConfig{
			Provider: CloudProviderAWS,
			Credentials: AWSConfig{
				AccessKey: "accessKey",
				SecretKey: "secretKey",
				Region:    "region",
				Bucket:    "bucket",
			},
		}

		buckt := setupBucktTest(t)

		// Cleanup to ensure the server is closed after the test
		t.Cleanup(func() {
			buckt.Close()
		})

		// set cloudService to nil to simulate an error
		buckt.cloudService = nil

		err := buckt.InitCloudService(config)
		assert.NoError(t, err)
		assert.NotNil(t, buckt.cloudService)

		// Add mock cloud service]
		buckt.cloudService = buckt.MockCloudService

		// Mock the expected behavior
		buckt.MockCloudService.On("UploadFolderToCloud", "user_1", "550e8400-e29b-41d4-a716-446655440000").
			Return(nil)

		// Call the method
		err = buckt.Buckt.TransferFolder("user_1", "550e8400-e29b-41d4-a716-446655440000")
		assert.NoError(t, err)

		// Verify expectations
		buckt.MockFolderService.AssertExpectations(t)
	})
}

// ✅ Test CloudProvider String() method
func TestCloudProviderString(t *testing.T) {
	tests := []struct {
		provider CloudProvider
		expected string
	}{
		{CloudProviderNone, "None"},
		{CloudProviderAWS, "AWS"},
		{CloudProviderAzure, "Azure"},
		{CloudProviderGCP, "GCP"},
		{CloudProvider(999), "None"}, // 🚨 Invalid provider
	}

	for _, test := range tests {
		if result := test.provider.String(); result != test.expected {
			t.Errorf("expected %s, got %s", test.expected, result)
		}
	}
}

// ✅ Test AWSConfig validation
func TestAWSConfigValidate(t *testing.T) {
	tests := []struct {
		config   AWSConfig
		expected error
	}{
		{AWSConfig{"accessKey", "secretKey", "region", "bucket"}, nil},
		{AWSConfig{"", "secretKey", "region", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "", "region", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "secretKey", "", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "secretKey", "region", ""}, fmt.Errorf("AWS credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test AzureConfig validation
func TestAzureConfigValidate(t *testing.T) {
	tests := []struct {
		config   AzureConfig
		expected error
	}{
		{AzureConfig{"accountName", "accountKey", "container"}, nil},
		{AzureConfig{"", "accountKey", "container"}, fmt.Errorf("AZURE credentials are incomplete")},
		{AzureConfig{"accountName", "", "container"}, fmt.Errorf("AZURE credentials are incomplete")},
		{AzureConfig{"accountName", "accountKey", ""}, fmt.Errorf("AZURE credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test GCPConfig validation
func TestGCPConfigValidate(t *testing.T) {
	tests := []struct {
		config   GCPConfig
		expected error
	}{
		{GCPConfig{"credentialsFile", "bucket"}, nil},
		{GCPConfig{"", "bucket"}, fmt.Errorf("GCP credentials are incomplete")},
		{GCPConfig{"credentialsFile", ""}, fmt.Errorf("GCP credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test NoCredentials validation
func TestNoCredentialsValidate(t *testing.T) {
	var creds NoCredentials
	if err := creds.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// ✅ Test InitCloudClient with valid and invalid inputs
func TestInitCloudClient(t *testing.T) {
	var fileService domain.FileService = mocks.NewMockFileService()
	var folderService domain.FolderService = mocks.NewMockFolderService()

	tests := []struct {
		config   CloudConfig
		expected error
	}{
		// ✅ Valid AWS Config
		{
			config: CloudConfig{
				Provider: CloudProviderAWS,
				Credentials: AWSConfig{
					AccessKey: "accessKey",
					SecretKey: "secretKey",
					Region:    "us-west-2",
					Bucket:    "my-bucket",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid AWS Config (missing AccessKey)
		{
			config: CloudConfig{
				Provider: CloudProviderAWS,
				Credentials: AWSConfig{
					AccessKey: "",
					SecretKey: "secretKey",
					Region:    "us-west-2",
					Bucket:    "my-bucket",
				},
			},
			expected: fmt.Errorf("AWS credentials are incomplete"),
		},
		// ✅ Valid Azure Config
		{
			config: CloudConfig{
				Provider: CloudProviderAzure,
				Credentials: AzureConfig{
					AccountName: "accountName",
					AccountKey:  "xYSz7q9wykiD7DPH/8tsukMQImrura/6MdwPdDR53D4=",
					Container:   "container",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid Azure Config (missing AccountName)
		{
			config: CloudConfig{
				Provider: CloudProviderAzure,
				Credentials: AzureConfig{
					AccountName: "",
					AccountKey:  "accountKey",
					Container:   "container",
				},
			},
			expected: fmt.Errorf("AZURE credentials are incomplete"),
		},
		// ✅ Valid GCP Config
		{
			config: CloudConfig{
				Provider: CloudProviderGCP,
				Credentials: GCPConfig{
					CredentialsFile: "internal/mocks/file.json",
					Bucket:          "bucket",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid GCP Config (missing CredentialsFile)
		{
			config: CloudConfig{
				Provider: CloudProviderGCP,
				Credentials: GCPConfig{
					CredentialsFile: "",
					Bucket:          "bucket",
				},
			},
			expected: fmt.Errorf("GCP credentials are incomplete"),
		},
		// 🚨 Invalid Cloud Provider
		{
			config: CloudConfig{
				Provider:    CloudProvider(999), // 🚨 Unknown provider
				Credentials: NoCredentials{},
			},
			expected: fmt.Errorf("local cloud service not implemented"),
		},
	}

	for _, test := range tests {
		_, err := initCloudClient(test.config, fileService, folderService)
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

type MockCredentials struct{}

func (m MockCredentials) Validate() error {
	return nil
}

func TestCloudConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name        string
		cloudConfig CloudConfig
		want        bool
	}{
		{
			name: "Empty CloudConfig",
			cloudConfig: CloudConfig{
				Provider:    CloudProviderNone,
				Credentials: nil,
			},
			want: true,
		},
		{
			name: "Non-empty CloudConfig with Provider",
			cloudConfig: CloudConfig{
				Provider:    CloudProviderAWS,
				Credentials: nil,
			},
			want: true,
		},
		{
			name: "Non-empty CloudConfig with Credentials",
			cloudConfig: CloudConfig{
				Provider:    CloudProviderNone,
				Credentials: &MockCredentials{},
			},
			want: true,
		},
		{
			name: "Non-empty CloudConfig with Provider and Credentials",
			cloudConfig: CloudConfig{
				Provider:    CloudProviderAWS,
				Credentials: &MockCredentials{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cloudConfig.isEmpty(); got != tt.want {
				t.Errorf("CloudConfig.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
