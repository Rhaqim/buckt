package model

import (
	"fmt"
	"reflect"
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

type CloudConfig struct {
	Provider    CloudProvider
	Credentials CloudCredentials
}

func (cc CloudConfig) IsEmpty() bool {
	return cc.Provider == CloudProviderNone || cc.Credentials == nil || (reflect.ValueOf(cc.Credentials).Kind() == reflect.Ptr && reflect.ValueOf(cc.Credentials).IsNil())
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
