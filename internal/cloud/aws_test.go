package cloud

import (
	"bytes"
	"context"
	"testing"

	"github.com/Rhaqim/buckt/internal/mocks"
	"github.com/Rhaqim/buckt/internal/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input)
	return &s3.PutObjectOutput{}, args.Error(1)
}

func TestNewAWSCloud(t *testing.T) {
	fileService := new(mocks.MockFileService)
	folderService := new(mocks.MockFolderService)

	awsCloud, err := NewAWSCloud("test-bucket", "us-west-2", fileService, folderService)

	assert.NoError(t, err)
	assert.NotNil(t, awsCloud)
	assert.Equal(t, "test-bucket", awsCloud.(*AWSCloud).BucketName)
}

func TestAWSCloud_UploadFile(t *testing.T) {
	mockS3Client := new(MockS3Client)
	fileService := new(mocks.MockFileService)
	folderService := new(mocks.MockFolderService)

	awsCloud := &AWSCloud{
		BucketName: "test-bucket",
		Client:     mockS3Client,
		BaseCloudStorage: BaseCloudStorage{
			ctx:                 context.Background(),
			FileService:         fileService,
			FolderService:       folderService,
			UploadFileFn:        nil,
			CreateEmptyFolderFn: nil,
		},
	}

	file := model.FileModel{
		Path:        "test/path",
		Data:        []byte("test data"),
		ContentType: "text/plain",
	}

	mockS3Client.On("PutObject", awsCloud.ctx, &s3.PutObjectInput{
		Bucket:      aws.String("test-bucket"),
		Key:         aws.String(file.Path),
		Body:        bytes.NewReader(file.Data),
		ContentType: aws.String(file.ContentType),
	}).Return(&s3.PutObjectOutput{}, nil)

	err := awsCloud.uploadFile(file.Name, file.ContentType, file.Path, file.Data, nil)
	assert.NoError(t, err)
	mockS3Client.AssertExpectations(t)
}

func TestAWSCloud_CreateEmptyFolder(t *testing.T) {
	mockS3Client := new(MockS3Client)
	fileService := new(mocks.MockFileService)
	folderService := new(mocks.MockFolderService)

	awsCloud := &AWSCloud{
		BucketName: "test-bucket",
		Client:     mockS3Client,
		BaseCloudStorage: BaseCloudStorage{
			ctx:                 context.Background(),
			FileService:         fileService,
			FolderService:       folderService,
			UploadFileFn:        nil,
			CreateEmptyFolderFn: nil,
		},
	}

	folderPath := "test/folder/"

	mockS3Client.On("PutObject", awsCloud.ctx, &s3.PutObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String(folderPath),
		Body:   bytes.NewReader([]byte{}),
	}).Return(&s3.PutObjectOutput{}, nil)

	err := awsCloud.createEmptyFolder(folderPath)
	assert.NoError(t, err)
	mockS3Client.AssertExpectations(t)
}
