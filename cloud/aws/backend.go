package aws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Backend struct {
	client     *s3.Client
	bucketName string
}

func NewBackend(conf AWSConfig) (*S3Backend, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	// Load AWS configuration (credentials, region, etc.)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(conf.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 	awsConfig := aws.Config{
	// 		Region:      cfg.Region,
	// 		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	// 	}

	// 	client := s3.NewFromConfig(awsConfig)

	client := s3.NewFromConfig(cfg)

	return &S3Backend{
		client:     client,
		bucketName: conf.Bucket,
	}, nil
}

func (s *S3Backend) Put(path string, data []byte) error {
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3Backend) Get(path string) ([]byte, error) {
	resp, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
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

func (s *S3Backend) Stream(path string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	// Caller must close the reader to avoid leaks
	return resp.Body, nil
}

func (s *S3Backend) Delete(path string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3Backend) Exists(path string) (bool, error) {
	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		var nsk *types.NotFound
		if ok := errorAs(err, &nsk); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Backend) Stat(path string) (*FileInfo, error) {
	resp, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	var size int64
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}
	return &FileInfo{
		Size:         size,
		LastModified: *resp.LastModified,
		ETag:         *resp.ETag,
		ContentType:  *resp.ContentType,
	}, nil
}

func (s *S3Backend) DeleteFolder(prefix string) error {
	// Step 1: List all objects under the prefix
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix + "/"), // Ensure trailing slash for "folder"
	})

	// Step 2: Batch delete
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		// Prepare delete objects request
		var objects []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}

		_, err = s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucketName),
			Delete: &types.Delete{Objects: objects},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
}

func (s3b *S3Backend) Move(oldPath, newPath string) error {
	// Copy
	_, err := s3b.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(s3b.bucketName),
		CopySource: aws.String(s3b.bucketName + "/" + oldPath),
		Key:        aws.String(newPath),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	waiter := s3.NewObjectExistsWaiter(s3b.client)

	maxWaitDuration := 30 * time.Second

	// Optional: Configure waiter options (e.g., max attempts, delay)
	waitOpts := func(o *s3.ObjectExistsWaiterOptions) {
		o.MinDelay = 5 * time.Second  // Default is 5 seconds
		o.MaxDelay = 10 * time.Second // Default is 5 seconds
	}

	// Wait for copy to finish
	err = waiter.Wait(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s3b.bucketName),
		Key:    aws.String(newPath),
	}, maxWaitDuration, waitOpts)
	if err != nil {
		return fmt.Errorf("failed waiting for copied object: %w", err)
	}

	// Delete old
	_, err = s3b.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s3b.bucketName),
		Key:    aws.String(oldPath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete old object: %w", err)
	}

	return nil
}

func errorAs(err error, target any) bool {
	return errors.As(err, target)
}
