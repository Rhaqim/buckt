package aws

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Rhaqim/buckt/internal/cloud"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type AWSCloud struct {
	cloud.BaseCloudStorage
	BucketName string
	Client     S3Client
}

func NewAWSCloud(cfg model.CloudConfig, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	// 	awsConfig := aws.Config{
	// 		Region:      cfg.Region,
	// 		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	// 	}

	// 	client := s3.NewFromConfig(awsConfig)

	creds, ok := cfg.Credentials.(model.AWSConfig)
	if !ok {
		return nil, fmt.Errorf("invalid AWS credentials")
	}

	if err := creds.Validate(); err != nil {
		return nil, err
	}

	region := config.WithRegion(creds.Region)

	awsConfig, err := config.LoadDefaultConfig(context.TODO(), region)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}
	client := s3.NewFromConfig(awsConfig)

	awsCloud := &AWSCloud{
		BucketName: creds.Bucket,
		Client:     client,
	}

	awsCloud.BaseCloudStorage = cloud.BaseCloudStorage{
		Ctx:                 context.Background(),
		FileService:         fileService,
		FolderService:       folderService,
		UploadFileFn:        awsCloud.uploadFile,
		CreateEmptyFolderFn: awsCloud.createEmptyFolder,
	}

	return awsCloud, nil
}

func (a *AWSCloud) uploadFile(file_name, content_type, file_path string, data []byte, metadata map[string]string) error {
	_, err := a.Client.PutObject(a.Ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.BucketName),
		Key:         aws.String(file_path),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(content_type),
		Metadata:    metadata,
	})

	return err
}

func (a *AWSCloud) createEmptyFolder(folderPath string) error {
	// Ensure folderPath ends with a slash to represent a folder in S3
	if folderPath[len(folderPath)-1] != '/' {
		folderPath += "/"
	}
	_, err := a.Client.PutObject(a.Ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.BucketName),
		Key:         aws.String(folderPath),
		Body:        bytes.NewReader([]byte{}),
		ContentType: aws.String("application/x-directory"),
		Metadata:    map[string]string{},
	})
	return err
}

func init() {
	cloud.RegisterCloudProvider(model.CloudProviderAWS, NewAWSCloud)
}
