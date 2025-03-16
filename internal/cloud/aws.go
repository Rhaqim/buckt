package cloud

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Rhaqim/buckt/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type AWSCloud struct {
	BaseCloudStorage
	BucketName string
	Client     S3Client
}

func NewAWSCloud(bucketName, region string, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	// 	awsConfig := aws.Config{
	// 		Region:      cfg.Region,
	// 		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	// 	}

	// 	client := s3.NewFromConfig(awsConfig)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}
	client := s3.NewFromConfig(cfg)

	awsCloud := &AWSCloud{
		BucketName: bucketName,
		Client:     client,
	}

	awsCloud.BaseCloudStorage = BaseCloudStorage{
		ctx:                 context.Background(),
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        awsCloud.uploadFile,
		CreateEmptyFolderFn: awsCloud.createEmptyFolder,
	}

	return awsCloud, nil
}

func (a *AWSCloud) uploadFile(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
	_, err := a.Client.PutObject(a.ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.BucketName),
		Key:         aws.String(file_path),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(content_type),
		Metadata:    metadata,
	})

	return err
}

func (a *AWSCloud) createEmptyFolder(folderPath string) error {
	_, err := a.Client.PutObject(a.ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.BucketName),
		Key:    aws.String(folderPath),
		Body:   bytes.NewReader([]byte{}),
	})
	return err
}
