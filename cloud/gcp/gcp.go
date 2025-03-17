package cloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/Rhaqim/buckt/internal/cloud"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"google.golang.org/api/option"
)

type GCPCloud struct {
	cloud.BaseCloudStorage
	BucketName string
	Client     *storage.Client
}

func NewGCPCloud(cfg model.CloudConfig, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	creds, ok := cfg.Credentials.(model.GCPConfig)
	if !ok {
		return nil, fmt.Errorf("invalid AWS credentials")
	}

	if err := creds.Validate(); err != nil {
		return nil, err
	}

	Ctx := context.Background()
	client, err := storage.NewClient(Ctx, option.WithCredentialsFile(creds.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP storage client: %v", err)
	}

	gcpCloud := &GCPCloud{
		BucketName: creds.Bucket,
		Client:     client,
	}

	gcpCloud.BaseCloudStorage = cloud.BaseCloudStorage{
		Ctx:                 Ctx,
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        gcpCloud.uploadFile,
		CreateEmptyFolderFn: gcpCloud.createEmptyFolder,
	}
	return gcpCloud, nil
}

func (g *GCPCloud) uploadFile(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
	bucket := g.Client.Bucket(g.BucketName)
	wc := bucket.Object(file_path).NewWriter(g.Ctx)
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
	wc := g.Client.Bucket(g.BucketName).Object(folderPath).NewWriter(g.Ctx)
	defer wc.Close()
	return nil
}

func init() {
	cloud.RegisterCloudProvider(model.CloudProviderGCP, NewGCPCloud)
}
