package cloud

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Rhaqim/buckt/internal/cloud"
	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAzureClient struct {
	mock.Mock
}

func (m *MockAzureClient) NewBlockBlobClient(blobName string) *blockblob.Client {
	args := m.Called(blobName)
	return args.Get(0).(*blockblob.Client)
}

func TestNewAzureCloud(t *testing.T) {
	mockFileService := new(mocks.MockFileService)
	mockFolderService := new(mocks.MockFolderService)

	creds := model.CloudConfig{
		Provider: model.CloudProviderAzure,
		Credentials: model.AzureConfig{
			AccountName: "accountName",
			AccountKey:  "accountKey",
			Container:   "containerName",
		},
	}

	azureCloud, err := NewAzureCloud(creds, mockFileService, mockFolderService)

	assert.NoError(t, err)
	assert.NotNil(t, azureCloud)
	assert.Equal(t, "containerName", azureCloud.(*AzureCloud).ContainerName)
}

func TestAzureCloud_uploadFile(t *testing.T) {
	mockClient := new(MockAzureClient)
	mockBlobClient := new(blockblob.Client)
	mockClient.On("NewBlockBlobClient", "test/path").Return(mockBlobClient)

	azureCloud := &AzureCloud{
		Client: mockClient,
		BaseCloudStorage: cloud.BaseCloudStorage{
			Ctx: context.Background(),
		},
	}

	file := model.FileModel{
		Path: "test/path",
		Data: []byte("test data"),
	}

	// mockBlobClient.On("UploadBuffer", mock.Anything, file.Data, mock.Anything).Return(nil, nil)

	err := azureCloud.uploadFile(file.Name, file.ContentType, file.Path, file.Data, nil)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	// mockBlobClient.AssertExpectations(t)
}

func TestAzureCloud_createEmptyFolder(t *testing.T) {
	mockClient := new(MockAzureClient)
	mockBlobClient := new(blockblob.Client)
	mockClient.On("NewBlockBlobClient", "test/folder/").Return(mockBlobClient)

	azureCloud := &AzureCloud{
		Client: mockClient,
		BaseCloudStorage: cloud.BaseCloudStorage{
			Ctx: context.Background(),
		},
	}

	// mockBlobClient.On("UploadBuffer", mock.Anything, []byte{}, mock.Anything).Return(nil, nil)

	err := azureCloud.createEmptyFolder("test/folder/")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	// mockBlobClient.AssertExpectations(t)
}
