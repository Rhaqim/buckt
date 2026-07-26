package azure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
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

func NewBackend(conf Config) (*AzureBackend, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	// Default to the public Azure Blob endpoint; allow an override for emulators
	// (Azurite) or private deployments.
	base := fmt.Sprintf("https://%s.blob.core.windows.net", conf.AccountName)
	if conf.Endpoint != "" {
		base = strings.TrimRight(conf.Endpoint, "/")
	}
	url := base + "/" + conf.Container

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

func (a *AzureBackend) Name() string {
	return "azure"
}

// Ping verifies connectivity to the Azure container by fetching its properties.
func (a *AzureBackend) Ping(ctx context.Context) error {
	_, err := a.client.GetProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("connectivity check failed for Azure container %q: %w", a.containerName, err)
	}
	return nil
}

func (a *AzureBackend) Put(ctx context.Context, path string, data []byte) error {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.UploadBuffer(ctx, data, &blockblob.UploadBufferOptions{})
	return err
}

func (a *AzureBackend) Get(ctx context.Context, path string) ([]byte, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *AzureBackend) List(ctx context.Context, prefix string) ([]string, error) {
	pager := a.client.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var paths []string
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name != nil {
				paths = append(paths, *blob.Name)
			}
		}
	}

	return paths, nil
}

func (a *AzureBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (a *AzureBackend) Delete(ctx context.Context, path string) error {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.Delete(ctx, nil)
	return err
}

func (a *AzureBackend) Exists(ctx context.Context, path string) (bool, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		// bloberror.HasCode is the SDK's way to match a service error code.
		// (bloberror.Code is a string, so errors.As against it panics.)
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *AzureBackend) Stat(ctx context.Context, path string) (*FileInfo, error) {
	blobClient := a.client.NewBlockBlobClient(path)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, err
	}

	fi := &FileInfo{}
	if props.ContentLength != nil {
		fi.Size = *props.ContentLength
	}
	if props.LastModified != nil {
		fi.LastModified = props.LastModified.UTC()
	}
	if props.ETag != nil {
		fi.ETag = string(*props.ETag)
	}
	if props.ContentType != nil {
		fi.ContentType = *props.ContentType
	}
	return fi, nil
}

func (a *AzureBackend) DeleteFolder(ctx context.Context, prefix string) error {
	pager := a.client.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			_ = a.Delete(ctx, *blob.Name) // best-effort delete
		}
	}

	return nil
}

func (a *AzureBackend) Move(ctx context.Context, oldPath, newPath string) error {
	// Apply a timeout if the parent context has none
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}

	srcBlob := a.client.NewBlockBlobClient(oldPath)
	destBlob := a.client.NewBlockBlobClient(newPath)

	_, err := destBlob.StartCopyFromURL(ctx, srcBlob.URL(), nil)
	if err != nil {
		return fmt.Errorf("failed to copy blob: %w", err)
	}

	// Poll copy completion with context cancellation
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("copy timed out for %s -> %s: %w", oldPath, newPath, ctx.Err())
		default:
		}

		props, err := destBlob.GetProperties(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get copy status: %w", err)
		}
		if props.CopyStatus != nil {
			switch *props.CopyStatus {
			case "success":
				return a.Delete(ctx, oldPath)
			case "failed", "aborted":
				return fmt.Errorf("blob copy %s: %s", *props.CopyStatus, safeDeref(props.CopyStatusDescription))
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
