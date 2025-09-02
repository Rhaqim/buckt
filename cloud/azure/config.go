package azure

import (
	"fmt"
	"time"
)

// AzureConfig implements CloudCredentials
type AzureConfig struct {
	AccountName string
	AccountKey  string
	Container   string
}

func (a AzureConfig) Validate() error {
	if a.AccountName == "" || a.AccountKey == "" || a.Container == "" {
		return fmt.Errorf("AZURE credentials are incomplete")
	}
	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}
