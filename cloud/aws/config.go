package aws

import (
	"fmt"
	"time"
)

// AWSConfig implements CloudCredentials
type AWSConfig struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
}

func (a AWSConfig) Validate() error {
	if a.AccessKey == "" || a.SecretKey == "" || a.Region == "" || a.Bucket == "" {
		return fmt.Errorf("AWS credentials are incomplete")
	}
	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
