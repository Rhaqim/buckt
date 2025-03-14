package cloud

import (
	"bytes"
	"context"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AWSCloud struct {
	BaseCloudStorage
	BucketName string
	Client     *s3.Client
}

func NewAWSCloud(bucketName, region string, fileService domain.FileService, folderService domain.FolderService) domain.CloudService {
	// 	awsConfig := aws.Config{
	// 		Region:      cfg.Region,
	// 		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	// 	}

	// 	client := s3.NewFromConfig(awsConfig)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
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
	return awsCloud
}

func (a *AWSCloud) uploadFile(file model.FileModel) error {
	_, err := a.Client.PutObject(a.ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.BucketName),
		Key:         aws.String(file.Path),
		Body:        bytes.NewReader(file.Data),
		ContentType: aws.String(file.ContentType),
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
