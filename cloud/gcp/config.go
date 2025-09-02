package gcp

import (
	"fmt"
	"time"
)

// GCPConfig implements CloudCredentials
type GCPConfig struct {
	CredentialsFile string
	Bucket          string
}

func (g GCPConfig) Validate() error {
	if g.CredentialsFile == "" || g.Bucket == "" {
		return fmt.Errorf("GCP credentials are incomplete")
	}
	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
