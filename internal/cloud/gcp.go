package cloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
	"google.golang.org/api/option"
)

type GCPCloud struct {
	BaseCloudStorage
	BucketName string
	Client     *storage.Client
}

func NewGCPCloud(credentialsFile, bucketName string, fileService domain.FileService, folderService domain.FolderService) domain.CloudService {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		panic(fmt.Sprintf("Failed to create GCP storage client: %v", err))
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
	return gcpCloud
}

func (g *GCPCloud) uploadFile(file model.FileModel) error {
	bucket := g.Client.Bucket(g.BucketName)
	wc := bucket.Object(file.Path).NewWriter(g.ctx)
	defer wc.Close()

	wc.ContentType = file.ContentType
	_, err := wc.Write(file.Data)
	return err
}

func (g *GCPCloud) createEmptyFolder(folderPath string) error {
	wc := g.Client.Bucket(g.BucketName).Object(folderPath).NewWriter(g.ctx)
	defer wc.Close()
	return nil
}
