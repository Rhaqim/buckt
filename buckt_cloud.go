package buckt

import (
	"fmt"

	"github.com/Rhaqim/buckt/internal/cloud"
	"github.com/Rhaqim/buckt/internal/domain"
)

type CloudProvider int

const (
	CloudProviderNone CloudProvider = iota
	CloudProviderAWS
	CloudProviderAzure
	CloudProviderGCP
)

func (cp CloudProvider) String() string {
	switch cp {
	case CloudProviderAWS:
		return "AWS"
	case CloudProviderAzure:
		return "Azure"
	case CloudProviderGCP:
		return "GCP"
	default:
		return "None"
	}
}

// CloudCredentials interface ensures every provider has a `Validate()` method
type CloudCredentials interface {
	Validate() error
}

type NoCredentials struct{}

func (n NoCredentials) Validate() error {
	return nil
}

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

func InitCloudClient(cfg CloudConfig, fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	// Validate provider configuration
	if err := cfg.Credentials.Validate(); err != nil {
		return nil, err
	}

	// Use type assertion to determine provider and create cloud service
	switch c := cfg.Credentials.(type) {
	case AWSConfig:
		return cloud.NewAWSCloud(c.Bucket, c.Region, fileService, folderService), nil
	case AzureConfig:
		return cloud.NewAzureCloud(c.AccountName, c.AccountKey, c.Container, fileService, folderService), nil
	case GCPConfig:
		return cloud.NewGCPCloud(c.CredentialsFile, c.Bucket, fileService, folderService), nil
	case NoCredentials:
		return NewLocalCloud(fileService, folderService)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", cfg.Provider.String())
	}
}

func NewLocalCloud(fileService domain.FileService, folderService domain.FolderService) (domain.CloudService, error) {
	return nil, fmt.Errorf("local cloud service not implemented")
}
