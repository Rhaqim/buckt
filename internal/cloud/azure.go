package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
)

type AzureCloud struct {
	BaseCloudStorage
	ContainerName string
	Client        *azblob.Client
}

func NewAzureCloud(accountName, accountKey, containerName string, fileService domain.FileService, folderService domain.FolderService) domain.CloudService {
	ctx := context.Background()
	url := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Azure credential: %v", err))
	}

	client, err := azblob.NewClientWithSharedKeyCredential(url, cred, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Azure storage client: %v", err))
	}

	azureCloud := &AzureCloud{
		ContainerName: containerName,
		Client:        client,
	}

	azureCloud.BaseCloudStorage = BaseCloudStorage{
		ctx:                 ctx,
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        azureCloud.uploadFile,
		CreateEmptyFolderFn: azureCloud.createEmptyFolder,
	}
	return azureCloud
}

func (a *AzureCloud) uploadFile(file model.FileModel) error {
	blobClient := a.Client.ServiceClient().NewContainerClient(a.ContainerName).NewBlockBlobClient(file.Path)
	_, err := blobClient.UploadBuffer(a.ctx, file.Data, &blockblob.UploadBufferOptions{})
	return err
}

func (a *AzureCloud) createEmptyFolder(folderPath string) error {
	blobClient := a.Client.ServiceClient().NewContainerClient(a.ContainerName).NewBlockBlobClient(folderPath)
	_, err := blobClient.UploadBuffer(a.ctx, []byte{}, &blockblob.UploadBufferOptions{})
	return err
}
