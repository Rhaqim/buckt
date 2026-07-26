package aws

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string

	// Optional: used when connecting to S3-compatible endpoints like Cloudflare R2, MinIO, etc.
	Endpoint     string // e.g., "https://<account_id>.r2.cloudflarestorage.com"
	UsePathStyle bool   // Set to true for path-style addressing (required for some S3-compatible services); auto-detected for Cloudflare R2 based on endpoint
}

func (a Config) Validate() error {
	var missing []string
	if a.AccessKey == "" {
		missing = append(missing, "AccessKey")
	}
	if a.SecretKey == "" {
		missing = append(missing, "SecretKey")
	}
	if a.Bucket == "" {
		missing = append(missing, "Bucket")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required field(s): %s", strings.Join(missing, ", "))
	}

	// Region is optional if a custom endpoint is provided (e.g., R2, MinIO)
	if a.Region == "" && a.Endpoint == "" {
		return fmt.Errorf("missing region or custom endpoint")
	}

	// Validate endpoint URL format if provided
	if a.Endpoint != "" {
		if _, err := url.ParseRequestURI(a.Endpoint); err != nil {
			return fmt.Errorf("invalid endpoint URL format: %w", err)
		}
	}

	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
