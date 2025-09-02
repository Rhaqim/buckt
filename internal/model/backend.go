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
	BackendProviderAWS
	BackendProviderAzure
	BackendProviderGCP
)

func (bp BackendProvider) IsValidProvider() bool {
	switch bp {
	case BackendProviderLocal,
		BackendProviderAWS,
		BackendProviderAzure,
		BackendProviderGCP:
		return true
	default:
		return false
	}
}

func (bp BackendProvider) String() string {
	switch bp {
	case BackendProviderAWS:
		return "AWS"
	case BackendProviderAzure:
		return "Azure"
	case BackendProviderGCP:
		return "GCP"
	default:
		return "Local"
	}
}
