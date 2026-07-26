package aws

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// These tests exercise the per-API-call metrics against a real S3-compatible
// endpoint (MinIO / Cloudflare R2). They are skipped unless BUCKT_MINIO_ENDPOINT
// is set. Note this file imports NO buckt package — the metrics hook is a
// stdlib-only callback, so backends stay independent of buckt.

func minioEnv(t *testing.T) (endpoint, key, secret, bucket string) {
	t.Helper()
	endpoint = os.Getenv("BUCKT_MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("BUCKT_MINIO_ENDPOINT not set; skipping S3-compatible metrics test")
	}
	key = getenv("BUCKT_MINIO_KEY", "minio")
	secret = getenv("BUCKT_MINIO_SECRET", "minio12345")
	bucket = getenv("BUCKT_MINIO_BUCKET", "buckt-test")
	return
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// ensureBucket creates the bucket (idempotently) using a raw client, since the
// backend itself never creates buckets.
func ensureBucket(t *testing.T, endpoint, key, secret, bucket string) {
	t.Helper()
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		config.WithRegion("us-east-1"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cl := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	// Best-effort create. Some S3-compatible endpoints reject SDK CreateBucket
	// via a custom endpoint resolver; the bucket is expected to already exist
	// (create it out-of-band, e.g. `mc mb`). "Already exists" is fine.
	_, err = cl.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err != nil {
		var owned *types.BucketAlreadyOwnedByYou
		var exists *types.BucketAlreadyExists
		if !errors.As(err, &owned) && !errors.As(err, &exists) {
			t.Logf("CreateBucket %q best-effort failed (assuming it exists): %v", bucket, err)
		}
	}
}

func TestS3Backend_PerAPIMetrics_MinIO(t *testing.T) {
	endpoint, key, secret, bucket := minioEnv(t)
	ensureBucket(t, endpoint, key, secret, bucket)

	be, err := NewBackend(Config{
		AccessKey:    key,
		SecretKey:    secret,
		Bucket:       bucket,
		Region:       "us-east-1",
		Endpoint:     endpoint,
		UsePathStyle: true,
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	// Install a local recorder — a plain stdlib-typed callback, no buckt import.
	var mu sync.Mutex
	counts := map[string]int{}
	be.SetOpRecorder(func(op string, _ int64, _ time.Duration, _ error) {
		mu.Lock()
		counts[op]++
		mu.Unlock()
	})

	ctx := context.Background()
	// Use a unique prefix so repeated runs / DeleteFolder don't collide.
	dir := "metrics-test"
	pathA := dir + "/a.txt"
	pathB := dir + "/b.txt"

	require := func(cond bool, format string, args ...any) {
		if !cond {
			t.Fatalf(format, args...)
		}
	}

	if err := be.Put(ctx, pathA, []byte("hello")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	ok, err := be.Exists(ctx, pathA)
	require(err == nil && ok, "Exists failed: ok=%v err=%v", ok, err)
	data, err := be.Get(ctx, pathA)
	require(err == nil && string(data) == "hello", "Get failed: %q err=%v", data, err)
	_, err = be.List(ctx, dir+"/")
	require(err == nil, "List failed: %v", err)
	require(be.Move(ctx, pathA, pathB) == nil, "Move failed")
	require(be.Delete(ctx, pathB) == nil, "Delete failed")
	// Put one more then DeleteFolder to exercise ListObjectsV2 + DeleteObjects.
	require(be.Put(ctx, dir+"/c.txt", []byte("x")) == nil, "Put c failed")
	require(be.DeleteFolder(ctx, dir) == nil, "DeleteFolder failed")

	mu.Lock()
	defer mu.Unlock()
	t.Logf("recorded S3 API calls: %v", counts)

	check := func(op string, min int) {
		if counts[op] < min {
			t.Errorf("expected >=%d %s calls, got %d", min, op, counts[op])
		}
	}
	check("s3:PutObject", 2)     // a.txt + c.txt
	check("s3:HeadObject", 1)    // Exists
	check("s3:GetObject", 1)     // Get
	check("s3:ListObjectsV2", 2) // List + DeleteFolder's paginator
	check("s3:CopyObject", 1)    // Move
	check("s3:DeleteObject", 2)  // Move's delete + Delete
	check("s3:DeleteObjects", 1) // DeleteFolder batch
}
