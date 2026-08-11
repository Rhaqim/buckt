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

	// rec, when set via SetOpRecorder, receives one record per real S3 API call
	// — including each ListObjectsV2 page and the CopyObject+DeleteObject a Move
	// issues — so callers can count exact Cloudflare R2 Class A/B operations.
	// nil means metrics are disabled.
	//
	// The signature uses only standard-library types on purpose: it lets buckt
	// inject a recorder here WITHOUT this module ever importing buckt, keeping
	// the dependency one-way (buckt depends on backends, never the reverse).
	rec func(op string, bytes int64, dur time.Duration, err error)
}

// Compile-time guard that S3Backend keeps the exact SetOpRecorder signature
// buckt detects structurally. If this stops compiling, the per-API-call metrics
// hook drifted and buckt would silently stop injecting the recorder.
var _ interface {
	SetOpRecorder(func(op string, bytes int64, dur time.Duration, err error))
} = (*S3Backend)(nil)

// SetOpRecorder installs a per-API-call metrics callback. buckt calls this
// (detected structurally) when its metrics are enabled; other callers may set it
// directly. Reported operation names are "s3:PutObject", "s3:GetObject",
// "s3:ListObjectsV2", etc. Safe to leave unset — metrics are then disabled.
func (s *S3Backend) SetOpRecorder(rec func(op string, bytes int64, dur time.Duration, err error)) {
	s.rec = rec
}

// record reports a single S3 API call. No-op when no recorder is set.
func (s *S3Backend) record(name string, bytes int64, start time.Time, err error) {
	if s.rec == nil {
		return
	}
	s.rec("s3:"+name, bytes, time.Since(start), err)
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

	// Handle custom endpoint (R2, MinIO, and other S3-compatible stores). Use
	// BaseEndpoint + UsePathStyle — the modern, Cloudflare-documented approach.
	// (A custom EndpointResolverV2 that returns only the base URL drops the
	// bucket from the request, which breaks path-style endpoints like MinIO with
	// "NoSuchBucket".)
	if conf.Endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(conf.Endpoint)
			o.UsePathStyle = conf.UsePathStyle || strings.HasSuffix(conf.Endpoint, CLOUDFLARE_R2_ENDPOINT_SUBSTRING)
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

// altKey returns the leading-slash-stripped form of a key and whether it differs
// from the input. buckt's nested-namespace paths are stored with a leading slash
// (e.g. "/user/root_folder/file"), but a local -> S3 migration copies objects
// under the keys the local backend's List() reports, which are relative and so
// have NO leading slash. Reads and deletes therefore try both forms so objects
// written by either path (direct/dual-write vs. bulk migration) resolve.
func altKey(path string) (string, bool) {
	alt := strings.TrimPrefix(path, "/")
	return alt, alt != path
}

// isNotFound reports whether err is an S3 "object missing" error (GetObject
// returns NoSuchKey; HeadObject returns NotFound).
func isNotFound(err error) bool {
	var ae smithy.APIError
	return errors.As(err, &ae) && (ae.ErrorCode() == "NoSuchKey" || ae.ErrorCode() == "NotFound")
}

// Ping verifies connectivity to the S3 bucket by performing a lightweight HeadBucket call.
// Call this after NewBackend to catch credential or network issues early.
func (s *S3Backend) Ping(ctx context.Context) error {
	start := time.Now()
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	s.record("HeadBucket", 0, start, err)
	if err != nil {
		return fmt.Errorf("S3 connectivity check failed for bucket %q: %w", s.bucketName, err)
	}
	return nil
}

func (s *S3Backend) Put(ctx context.Context, path string, data []byte) error {
	start := time.Now()
	err := withRetry(ctx, 3, func() error {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(path),
			Body:   bytes.NewReader(data),
		})
		return err
	})
	s.record("PutObject", int64(len(data)), start, err)
	return err
}

func (s *S3Backend) Get(ctx context.Context, path string) ([]byte, error) {
	data, err := s.getObject(ctx, path)
	if err != nil && isNotFound(err) {
		if alt, ok := altKey(path); ok {
			if d, altErr := s.getObject(ctx, alt); altErr == nil {
				return d, nil
			}
		}
	}
	return data, err
}

func (s *S3Backend) getObject(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		s.record("GetObject", 0, start, err)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	buf := new(bytes.Buffer)
	_, copyErr := io.Copy(buf, resp.Body)
	s.record("GetObject", int64(buf.Len()), start, copyErr)
	if copyErr != nil {
		return nil, copyErr
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
		start := time.Now()
		page, err := paginator.NextPage(ctx)
		s.record("ListObjectsV2", 0, start, err)
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
	rc, err := s.streamObject(ctx, path)
	if err != nil && isNotFound(err) {
		if alt, ok := altKey(path); ok {
			if r, altErr := s.streamObject(ctx, alt); altErr == nil {
				return r, nil
			}
		}
	}
	return rc, err
}

func (s *S3Backend) streamObject(ctx context.Context, key string) (io.ReadCloser, error) {
	start := time.Now()
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	s.record("GetObject", 0, start, err)
	if err != nil {
		return nil, err
	}
	// Caller must close the reader to avoid leaks
	return resp.Body, nil
}

// Delete removes the object. It deletes both the given key and its
// leading-slash-stripped form so a file migrated under the relative key is
// removed too (deletes are idempotent, so the redundant call is harmless).
func (s *S3Backend) Delete(ctx context.Context, path string) error {
	err := s.deleteKey(ctx, path)
	if alt, ok := altKey(path); ok {
		if altErr := s.deleteKey(ctx, alt); altErr != nil && err == nil {
			err = altErr
		}
	}
	return err
}

func (s *S3Backend) deleteKey(ctx context.Context, key string) error {
	start := time.Now()
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	s.record("DeleteObject", 0, start, err)
	return err
}

func (s *S3Backend) Exists(ctx context.Context, path string) (bool, error) {
	exists, err := s.headExists(ctx, path)
	if err == nil && !exists {
		if alt, ok := altKey(path); ok {
			return s.headExists(ctx, alt)
		}
	}
	return exists, err
}

func (s *S3Backend) headExists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	s.record("HeadObject", 0, start, err)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Backend) Stat(ctx context.Context, path string) (*FileInfo, error) {
	start := time.Now()
	resp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	s.record("HeadObject", 0, start, err)
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
	// Respect parent context's existing deadline
	ctx, cancel := withTimeoutIfNone(ctx, MOVE_TIMEOUT)
	defer cancel()

	// Copy from whichever key form the source object actually lives under —
	// direct writes keep the leading slash, a migration copy dropped it.
	err := s3b.copyObject(ctx, oldPath, newPath)
	if err != nil && isNotFound(err) {
		if alt, ok := altKey(oldPath); ok {
			err = s3b.copyObject(ctx, alt, newPath)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Remove the old object under both key forms (Delete handles both).
	if delErr := s3b.Delete(ctx, oldPath); delErr != nil {
		// Log but don't fail — the copy succeeded, the old object is just orphaned.
		log.Printf("warning: failed to delete old object %s after move: %v\n", oldPath, delErr)
	}

	return nil
}

func (s3b *S3Backend) copyObject(ctx context.Context, src, dst string) error {
	start := time.Now()
	_, err := s3b.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s3b.bucketName),
		CopySource: aws.String(s3b.bucketName + "/" + src),
		Key:        aws.String(dst),
	})
	s3b.record("CopyObject", 0, start, err)
	return err
}

func (s *S3Backend) DeleteFolder(ctx context.Context, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	base := strings.TrimSuffix(prefix, "/") + "/"
	if err := s.deletePrefix(ctx, base); err != nil {
		return err
	}
	// Also clear objects a migration wrote under the leading-slash-stripped key.
	if alt, ok := altKey(base); ok {
		if err := s.deletePrefix(ctx, alt); err != nil {
			return err
		}
	}
	return nil
}

func (s *S3Backend) deletePrefix(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})

	const batchSize = 1000
	var batch []types.ObjectIdentifier

	for paginator.HasMorePages() {
		start := time.Now()
		page, err := paginator.NextPage(ctx)
		s.record("ListObjectsV2", 0, start, err)
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
	start := time.Now()
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{Objects: objects},
	})
	s.record("DeleteObjects", int64(len(objects)), start, err)
	if err != nil {
		return fmt.Errorf("failed to delete batch (%d items): %w", len(objects), err)
	}
	return nil
}

func withRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

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

func withTimeoutIfNone(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
