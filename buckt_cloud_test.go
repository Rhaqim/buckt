package buckt

import (
	"fmt"
	"testing"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/mocks"
)

// ✅ Test CloudProvider String() method
func TestCloudProviderString(t *testing.T) {
	tests := []struct {
		provider CloudProvider
		expected string
	}{
		{CloudProviderNone, "None"},
		{CloudProviderAWS, "AWS"},
		{CloudProviderAzure, "Azure"},
		{CloudProviderGCP, "GCP"},
		{CloudProvider(999), "None"}, // 🚨 Invalid provider
	}

	for _, test := range tests {
		if result := test.provider.String(); result != test.expected {
			t.Errorf("expected %s, got %s", test.expected, result)
		}
	}
}

// ✅ Test AWSConfig validation
func TestAWSConfigValidate(t *testing.T) {
	tests := []struct {
		config   AWSConfig
		expected error
	}{
		{AWSConfig{"accessKey", "secretKey", "region", "bucket"}, nil},
		{AWSConfig{"", "secretKey", "region", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "", "region", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "secretKey", "", "bucket"}, fmt.Errorf("AWS credentials are incomplete")},
		{AWSConfig{"accessKey", "secretKey", "region", ""}, fmt.Errorf("AWS credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test AzureConfig validation
func TestAzureConfigValidate(t *testing.T) {
	tests := []struct {
		config   AzureConfig
		expected error
	}{
		{AzureConfig{"accountName", "accountKey", "container"}, nil},
		{AzureConfig{"", "accountKey", "container"}, fmt.Errorf("AZURE credentials are incomplete")},
		{AzureConfig{"accountName", "", "container"}, fmt.Errorf("AZURE credentials are incomplete")},
		{AzureConfig{"accountName", "accountKey", ""}, fmt.Errorf("AZURE credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test GCPConfig validation
func TestGCPConfigValidate(t *testing.T) {
	tests := []struct {
		config   GCPConfig
		expected error
	}{
		{GCPConfig{"credentialsFile", "bucket"}, nil},
		{GCPConfig{"", "bucket"}, fmt.Errorf("GCP credentials are incomplete")},
		{GCPConfig{"credentialsFile", ""}, fmt.Errorf("GCP credentials are incomplete")},
	}

	for _, test := range tests {
		err := test.config.Validate()
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}

// ✅ Test NoCredentials validation
func TestNoCredentialsValidate(t *testing.T) {
	var creds NoCredentials
	if err := creds.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// ✅ Test InitCloudClient with valid and invalid inputs
func TestInitCloudClient(t *testing.T) {
	var fileService domain.FileService = mocks.NewMockFileService()
	var folderService domain.FolderService = mocks.NewMockFolderService()

	tests := []struct {
		config   CloudConfig
		expected error
	}{
		// ✅ Valid AWS Config
		{
			config: CloudConfig{
				Provider: CloudProviderAWS,
				Credentials: AWSConfig{
					AccessKey: "accessKey",
					SecretKey: "secretKey",
					Region:    "us-west-2",
					Bucket:    "my-bucket",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid AWS Config (missing AccessKey)
		{
			config: CloudConfig{
				Provider: CloudProviderAWS,
				Credentials: AWSConfig{
					AccessKey: "",
					SecretKey: "secretKey",
					Region:    "us-west-2",
					Bucket:    "my-bucket",
				},
			},
			expected: fmt.Errorf("AWS credentials are incomplete"),
		},
		// ✅ Valid Azure Config
		{
			config: CloudConfig{
				Provider: CloudProviderAzure,
				Credentials: AzureConfig{
					AccountName: "accountName",
					AccountKey:  "accountKey",
					Container:   "container",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid Azure Config (missing AccountName)
		{
			config: CloudConfig{
				Provider: CloudProviderAzure,
				Credentials: AzureConfig{
					AccountName: "",
					AccountKey:  "accountKey",
					Container:   "container",
				},
			},
			expected: fmt.Errorf("AZURE credentials are incomplete"),
		},
		// ✅ Valid GCP Config
		{
			config: CloudConfig{
				Provider: CloudProviderGCP,
				Credentials: GCPConfig{
					CredentialsFile: "file.json",
					Bucket:          "bucket",
				},
			},
			expected: nil,
		},
		// 🚨 Invalid GCP Config (missing CredentialsFile)
		{
			config: CloudConfig{
				Provider: CloudProviderGCP,
				Credentials: GCPConfig{
					CredentialsFile: "",
					Bucket:          "bucket",
				},
			},
			expected: fmt.Errorf("GCP credentials are incomplete"),
		},
		// 🚨 Invalid Cloud Provider
		{
			config: CloudConfig{
				Provider:    CloudProvider(999), // 🚨 Unknown provider
				Credentials: NoCredentials{},
			},
			expected: fmt.Errorf("local cloud service not implemented"),
		},
	}

	for _, test := range tests {
		_, err := InitCloudClient(test.config, fileService, folderService)
		if err == nil && test.expected != nil || err != nil && test.expected == nil || err != nil && test.expected != nil && err.Error() != test.expected.Error() {
			t.Errorf("expected %v, got %v", test.expected, err)
		}
	}
}
