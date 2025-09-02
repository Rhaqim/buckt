package azure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type AzureBackend struct {
	client        *container.Client
	containerName string
}

func NewAzureBackend(conf AzureConfig) (*AzureBackend, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/%s", conf.AccountName, conf.Container)

	cred, err := azblob.NewSharedKeyCredential(conf.AccountName, conf.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	client, err := container.NewClientWithSharedKeyCredential(url, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure container client: %w", err)
	}

	return &AzureBackend{
		client:        client,
		containerName: conf.Container,
	}, nil
}

func (a *AzureBackend) Put(path string, data []byte) error {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.UploadBuffer(context.TODO(), data, &blockblob.UploadBufferOptions{})
	return err
}

func (a *AzureBackend) Get(path string) ([]byte, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	resp, err := blobClient.DownloadStream(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *AzureBackend) Stream(path string) (io.ReadCloser, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	resp, err := blobClient.DownloadStream(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (a *AzureBackend) Delete(path string) error {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.Delete(context.TODO(), nil)
	return err
}

func (a *AzureBackend) Exists(path string) (bool, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.GetProperties(context.TODO(), nil)
	if err != nil {
		var respErr bloberror.Code
		if ok := errorAs(err, &respErr); ok && respErr == bloberror.BlobNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *AzureBackend) Stat(path string) (*FileInfo, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	props, err := blobClient.GetProperties(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Size:         *props.ContentLength,
		LastModified: props.LastModified.UTC(),
		ETag:         string(*props.ETag),
		ContentType:  *props.ContentType,
	}, nil
}

func (a *AzureBackend) DeleteFolder(prefix string) error {
	pager := a.client.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		page, err := pager.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			_ = a.Delete(*blob.Name) // best-effort delete
		}
	}

	return nil
}

func (a *AzureBackend) Move(oldPath, newPath string) error {
	// Copy
	srcBlob := a.client.NewBlockBlobClient(oldPath)
	destBlob := a.client.NewBlockBlobClient(newPath)

	_, err := destBlob.StartCopyFromURL(context.TODO(), srcBlob.URL(), nil)
	if err != nil {
		return fmt.Errorf("failed to copy blob: %w", err)
	}

	// Poll copy completion
	for {
		props, err := destBlob.GetProperties(context.TODO(), nil)
		if err != nil {
			return fmt.Errorf("failed to get copy status: %w", err)
		}
		if props.CopyStatus != nil && *props.CopyStatus == "success" {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Delete old
	return a.Delete(oldPath)
}

func errorAs(err error, target any) bool {
	return errors.As(err, target)
}
