package gcp

import (
	"fmt"
	"time"
)

type Config struct {
	CredentialsFile string
	Bucket          string
}

func (g Config) Validate() error {
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
