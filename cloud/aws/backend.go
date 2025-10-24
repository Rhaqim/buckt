package aws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/cenkalti/backoff/v4"
)

type S3Backend struct {
	client     *s3.Client
	bucketName string
}

func NewBackend(conf Config) (*S3Backend, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	opts := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(conf.AccessKey, conf.SecretKey, "")),
	}

	// Region: required for AWS, optional for R2
	if conf.Region != "" {
		opts = append(opts, config.WithRegion(conf.Region))
	} else {
		opts = append(opts, config.WithRegion(AUTO_REGION))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS/R2 config: %w", err)
	}

	var client *s3.Client

	// Handle custom endpoint (R2, MinIO, etc.)
	if conf.Endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = conf.UsePathStyle || strings.HasSuffix(conf.Endpoint, CLOUDFLARE_R2_ENDPOINT_SUBSTRING)
			o.EndpointResolverV2 = &customEndpointResolver{rawURL: conf.Endpoint}
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return &S3Backend{
		client:     client,
		bucketName: conf.Bucket,
	}, nil
}

func (s *S3Backend) Name() string {
	return NAME
}

func (s *S3Backend) Put(ctx context.Context, path string, data []byte) error {
	err := withRetry(ctx, 3, func() error {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(path),
			Body:   bytes.NewReader(data),
		})
		return err
	})

	return err
}

func (s *S3Backend) Get(ctx context.Context, path string) ([]byte, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
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

func (s *S3Backend) List(ctx context.Context, prefix string) ([]string, error) {
	var paths []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			paths = append(paths, *obj.Key)
		}
	}

	return paths, nil
}

func (s *S3Backend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	// Caller must close the reader to avoid leaks
	return resp.Body, nil
}

func (s *S3Backend) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	return err
}

func (s *S3Backend) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "NotFound" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Backend) Stat(ctx context.Context, path string) (*FileInfo, error) {
	resp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}

	fi := &FileInfo{}
	if resp.ContentLength != nil {
		fi.Size = *resp.ContentLength
	}
	if resp.LastModified != nil {
		fi.LastModified = *resp.LastModified
	}
	if resp.ETag != nil {
		fi.ETag = *resp.ETag
	}
	if resp.ContentType != nil {
		fi.ContentType = *resp.ContentType
	}
	return fi, nil
}
func (s3b *S3Backend) Move(ctx context.Context, oldPath, newPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := s3b.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s3b.bucketName),
		CopySource: aws.String(s3b.bucketName + "/" + oldPath),
		Key:        aws.String(newPath),
	})
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Asynchronous best-effort delete
	go func(bucket, key string) {
		delCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, delErr := s3b.client.DeleteObject(delCtx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if delErr != nil {
			log.Printf("async delete failed for %s: %v\n", key, delErr)
			// Optionally enqueue for retry
			// s3b.cleanupQueue.Enqueue(key)
		}
	}(s3b.bucketName, oldPath)

	return nil
}

func (s *S3Backend) DeleteFolder(ctx context.Context, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(strings.TrimSuffix(prefix, "/") + "/"),
	})

	const batchSize = 1000
	var batch []types.ObjectIdentifier

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			batch = append(batch, types.ObjectIdentifier{Key: obj.Key})
			if len(batch) == batchSize {
				if err := s.deleteBatch(ctx, batch); err != nil {
					return err
				}
				batch = batch[:0]
			}
		}
	}

	if len(batch) > 0 {
		if err := s.deleteBatch(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (s *S3Backend) deleteBatch(ctx context.Context, objects []types.ObjectIdentifier) error {
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{Objects: objects},
	})
	if err != nil {
		return fmt.Errorf("failed to delete batch (%d items): %w", len(objects), err)
	}
	return nil
}

func withRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 200 * time.Millisecond
	b.MaxElapsedTime = 0 // disable total timeout; respect context instead

	return backoff.RetryNotify(
		func() error {
			err := fn()
			if err == nil {
				return nil
			}

			// Retry on transient network-level errors
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return err
			}

			// Retry on syscall-level errors like connection refused or reset
			if errors.Is(err, syscall.ECONNRESET) ||
				errors.Is(err, syscall.ECONNREFUSED) ||
				errors.Is(err, syscall.ETIMEDOUT) {
				return err
			}

			var apiErr smithy.APIError
			if errors.As(err, &apiErr) {
				status := 0
				if h, ok := apiErr.(interface{ HTTPStatusCode() int }); ok {
					status = h.HTTPStatusCode()
				}

				// Retry only for 5xx
				if status >= 500 {
					return err
				}
			}

			// Non-retryable error → stop immediately
			return backoff.Permanent(err)
		},
		backoff.WithContext(backoff.WithMaxRetries(b, uint64(maxAttempts)), ctx),
		func(err error, next time.Duration) {
			log.Printf("Retrying after %v: %v", next, err)
		},
	)
}
