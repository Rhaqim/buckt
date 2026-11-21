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

// TestPresignGetURLWithHeaders checks the response-header overrides are part of
// the signed URL, so the store serves our Content-Type / Disposition /
// Cache-Control instead of the object's stored metadata.
func TestPresignGetURLWithHeaders(t *testing.T) {
	be, err := NewBackend(Config{
		AccessKey: "AKIAEXAMPLE",
		SecretKey: "secretexamplekey",
		Bucket:    "my-bucket",
		Endpoint:  "https://accountid.r2.cloudflarestorage.com",
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	url, err := be.PresignGetURLWithHeaders(context.Background(), "voice.m4a", 15*time.Minute,
		"audio/mp4", "inline; filename*=UTF-8''voice.m4a", "private, max-age=900")
	if err != nil {
		t.Fatalf("PresignGetURLWithHeaders: %v", err)
	}
	for _, want := range []string{
		"response-content-type=audio%2Fmp4",
		"response-content-disposition=inline",
		"response-cache-control=private",
		"X-Amz-Signature",
	} {
		if !strings.Contains(url, want) {
			t.Errorf("presigned GET URL missing %q\n url = %s", want, url)
		}
	}

	// Empty values must not add overrides (plain PresignGetURL behaviour).
	plain, err := be.PresignGetURLWithHeaders(context.Background(), "voice.m4a", time.Minute, "", "", "")
	if err != nil {
		t.Fatalf("PresignGetURLWithHeaders: %v", err)
	}
	if strings.Contains(plain, "response-") {
		t.Errorf("empty overrides should add no response-* params\n url = %s", plain)
	}
}

func TestContentTypeFor(t *testing.T) {
	cases := []struct {
		key  string
		data []byte
		want string
	}{
		{"u/voice.m4a", []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'M', '4', 'A', ' '}, "audio/mp4"},
		{"u/song.MP3", []byte{0xFF, 0xFB, 0x90, 0x64}, "audio/mpeg"},
		{"u/rec.webm", []byte{0x1A, 0x45, 0xDF, 0xA3}, "video/webm"},
		{"u/logo.svg", []byte("<svg></svg>"), "image/svg+xml"},
		{"u/photo.jpg", nil, "image/jpeg"},
		// No extension (e.g. a derivative key): fall back to sniffing.
		{"_derivatives/abc/thumbnail", []byte("\x89PNG\r\n\x1a\n"), "image/png"},
		{"blob", []byte{0x00, 0x01, 0x02}, "application/octet-stream"},
	}
	for _, c := range cases {
		if got := contentTypeFor(c.key, c.data); got != c.want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", c.key, got, c.want)
		}
	}
}
