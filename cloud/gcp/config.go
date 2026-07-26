package gcp

import (
	"fmt"
	"time"
)

type Config struct {
	CredentialsFile string
	Bucket          string

	// Endpoint optionally overrides the GCS API endpoint. Leave empty for the
	// real service. Set it to point at an emulator (fake-gcs-server), e.g.
	// "http://localhost:4443/storage/v1/"; authentication is then skipped and
	// CredentialsFile may be empty.
	Endpoint string
}

func (g Config) Validate() error {
	if g.Bucket == "" {
		return fmt.Errorf("GCP bucket is required")
	}
	// Credentials are required for the real service, but an Endpoint override
	// (emulator/private) authenticates differently, so allow it without a file.
	if g.CredentialsFile == "" && g.Endpoint == "" {
		return fmt.Errorf("GCP requires a CredentialsFile (or an Endpoint override)")
	}
	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
