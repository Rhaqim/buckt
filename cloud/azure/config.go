package azure

import (
	"fmt"
	"time"
)

type Config struct {
	AccountName string
	AccountKey  string
	Container   string

	// Endpoint optionally overrides the blob service URL. Leave empty for the
	// real Azure Blob service (https://<account>.blob.core.windows.net). Set it
	// to point at an emulator or private deployment, e.g. Azurite:
	// "http://127.0.0.1:10000/devstoreaccount1". The container name is appended.
	Endpoint string
}

func (a Config) Validate() error {
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
