package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Rhaqim/buckt/internal/domain"
)

type AzureClient interface {
	NewBlockBlobClient(blobName string) *blockblob.Client
}

type AzureCloud struct {
	BaseCloudStorage
	ContainerName string
	Client        AzureClient
}

func NewAzureCloud(accountName, accountKey, containerName string, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	ctx := context.Background()
	url := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %v", err)
	}

	client, err := azblob.NewClientWithSharedKeyCredential(url, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure storage client: %v", err)
	}

	azureCloud := &AzureCloud{
		ContainerName: containerName,
		Client:        client.ServiceClient().NewContainerClient(containerName),
	}

	azureCloud.BaseCloudStorage = BaseCloudStorage{
		ctx:                 ctx,
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        azureCloud.uploadFile,
		CreateEmptyFolderFn: azureCloud.createEmptyFolder,
	}
	return azureCloud, nil
}

func (a *AzureCloud) uploadFile(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
	blobClient := a.Client.NewBlockBlobClient(file_path)

	convertedMetadata := make(map[string]*string)
	for k, v := range metadata {
		val := v
		convertedMetadata[k] = &val
	}
	_, err := blobClient.UploadBuffer(a.ctx, data, &blockblob.UploadBufferOptions{
		Metadata: convertedMetadata,
	})
	return err
}

func (a *AzureCloud) createEmptyFolder(folderPath string) error {
	blobClient := a.Client.NewBlockBlobClient(folderPath)
	_, err := blobClient.UploadBuffer(a.ctx, []byte{}, &blockblob.UploadBufferOptions{})
	return err
}
