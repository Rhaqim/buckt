package cloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/Rhaqim/buckt/internal/domain"
	"google.golang.org/api/option"
)

type GCPCloud struct {
	BaseCloudStorage
	BucketName string
	Client     *storage.Client
}

func NewGCPCloud(credentialsFile, bucketName string, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %v", err)
	}

	gcpCloud := &GCPCloud{
		BucketName: bucketName,
		Client:     client,
	}

	gcpCloud.BaseCloudStorage = BaseCloudStorage{
		ctx:                 ctx,
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        gcpCloud.uploadFile,
		CreateEmptyFolderFn: gcpCloud.createEmptyFolder,
	}
	return gcpCloud, nil
}

func (g *GCPCloud) uploadFile(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
	bucket := g.Client.Bucket(g.BucketName)
	wc := bucket.Object(file_path).NewWriter(g.ctx)
	defer wc.Close()

	objectAttrs := storage.ObjectAttrs{
		ContentType: content_type,
		Name:        file_name,
		Metadata:    metadata,
	}

	wc.ObjectAttrs = objectAttrs
	wc.ContentType = content_type
	_, err := wc.Write(data)
	return err
}

func (g *GCPCloud) createEmptyFolder(folderPath string) error {
	wc := g.Client.Bucket(g.BucketName).Object(folderPath).NewWriter(g.ctx)
	defer wc.Close()
	return nil
}
