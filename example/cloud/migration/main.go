// Package main demonstrates migration from local filesystem to AWS S3 storage using the buckt library.
// This example shows how to configure and initialize a buckt client with AWS S3 as a secondary backend
// for cloud storage migration scenarios.
//
// The program sets up an AWS S3 backend with the necessary credentials and configuration,
// registers it as a secondary backend with buckt, and initializes a default client
// with logging capabilities enabled.
//
// Usage:
//   - Replace the placeholder AWS credentials with actual values
//   - Ensure the specified S3 bucket exists and is accessible
//   - Pass in the enabled migration configuration to support dual-write operations
//   - Run the program to establish a connection to AWS S3 through buckt
//
// Prerequisites:
//   - Valid AWS credentials (Access Key and Secret Key)
//   - Existing S3 bucket with appropriate permissions
//   - Network connectivity to AWS S3 services
package main

import (
	"fmt"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/aws"
)

func main() {

	cloudConfig := aws.Config{
		AccessKey: "accessKey",
		SecretKey: "secretKey",
		Region:    "us-west-2",
		Bucket:    "my-bucket",
	}

	// Initialize AWS S3 backend
	awsBackend, err := aws.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create AWS backend:", err)
		return
	}

	// Enable migration mode
	migration := buckt.EnableMigration()

	// Register AWS S3 as the secondary backend for migration target
	backend := buckt.RegisterSecondaryBackend(awsBackend)

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend, migration)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
