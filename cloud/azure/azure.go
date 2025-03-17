package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Rhaqim/buckt/internal/cloud"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
)

type AzureClient interface {
	NewBlockBlobClient(blobName string) *blockblob.Client
}

type AzureCloud struct {
	cloud.BaseCloudStorage
	ContainerName string
	Client        AzureClient
}

func NewAzureCloud(cfg model.CloudConfig, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {

	creds, ok := cfg.Credentials.(model.AzureConfig)
	if !ok {
		return nil, fmt.Errorf("invalid AWS credentials")
	}

	if err := creds.Validate(); err != nil {
		return nil, err
	}

	Ctx := context.Background()
	url := fmt.Sprintf("https://%s.blob.core.windows.net", creds.AccountName)

	cred, err := azblob.NewSharedKeyCredential(creds.AccountName, creds.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %v", err)
	}

	client, err := azblob.NewClientWithSharedKeyCredential(url, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure storage client: %v", err)
	}

	azureCloud := &AzureCloud{
		ContainerName: creds.Container,
		Client:        client.ServiceClient().NewContainerClient(creds.Container),
	}

	azureCloud.BaseCloudStorage = cloud.BaseCloudStorage{
		Ctx:                 Ctx,
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
	_, err := blobClient.UploadBuffer(a.Ctx, data, &blockblob.UploadBufferOptions{
		Metadata: convertedMetadata,
	})
	return err
}

func (a *AzureCloud) createEmptyFolder(folderPath string) error {
	blobClient := a.Client.NewBlockBlobClient(folderPath)
	_, err := blobClient.UploadBuffer(a.Ctx, []byte{}, &blockblob.UploadBufferOptions{})
	return err
}

func init() {
	cloud.RegisterCloudProvider(model.CloudProviderAzure, NewAzureCloud)
}
