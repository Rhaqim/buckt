package model

import "time"

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}

type BackendProvider int

const (
	BackendProviderLocal BackendProvider = iota
	BackendProviderS3
	BackendProviderAzure
	BackendProviderGCP
	BackendProviderCustom
)

func (bp BackendProvider) IsValidProvider() bool {
	switch bp {
	case BackendProviderLocal,
		BackendProviderS3,
		BackendProviderAzure,
		BackendProviderGCP,
		BackendProviderCustom:
		return true
	default:
		return false
	}
}

func (bp BackendProvider) String() string {
	switch bp {
	case BackendProviderS3:
		return "S3"
	case BackendProviderAzure:
		return "Azure"
	case BackendProviderGCP:
		return "GCP"
	case BackendProviderCustom:
		return "Custom"
	default:
		return "Local"
	}
}
