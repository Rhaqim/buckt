package aws

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestPresignGetURL signs a flat key, which needs no HEAD and therefore no
// network: SigV4 presigning computes the signature locally. It checks the URL
// targets the right bucket/key and carries the expected query parameters.
func TestPresignGetURL(t *testing.T) {
	be, err := NewBackend(Config{
		AccessKey: "AKIAEXAMPLE",
		SecretKey: "secretexamplekey",
		Bucket:    "my-bucket",
		Endpoint:  "https://accountid.r2.cloudflarestorage.com",
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	url, err := be.PresignGetURL(context.Background(), "photo.jpg", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGetURL: %v", err)
	}

	for _, want := range []string{
		"my-bucket",         // path-style bucket segment (R2 auto path-style)
		"photo.jpg",         // the object key
		"X-Amz-Signature",   // SigV4 signature present
		"X-Amz-Expires=900", // ttl in seconds
		"X-Amz-Credential",  // credential scope present
	} {
		if !strings.Contains(url, want) {
			t.Errorf("presigned GET URL missing %q\n url = %s", want, url)
		}
	}
}

// TestPresignPutURL checks the upload URL is a signed PUT for the given key.
func TestPresignPutURL(t *testing.T) {
	be, err := NewBackend(Config{
		AccessKey: "AKIAEXAMPLE",
		SecretKey: "secretexamplekey",
		Bucket:    "my-bucket",
		Endpoint:  "https://accountid.r2.cloudflarestorage.com",
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	url, err := be.PresignPutURL(context.Background(), "uploads/report.pdf", 10*time.Minute)
	if err != nil {
		t.Fatalf("PresignPutURL: %v", err)
	}

	for _, want := range []string{
		"my-bucket",
		"uploads/report.pdf",
		"X-Amz-Signature",
		"X-Amz-Expires=600",
	} {
		if !strings.Contains(url, want) {
			t.Errorf("presigned PUT URL missing %q\n url = %s", want, url)
		}
	}
}
