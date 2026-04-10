// Package main demonstrates migration from local filesystem to AWS S3 storage.
//
// This example shows how to set up a dual-write migration: writes go to both
// the local filesystem and S3, and reads fall back to S3 when the local copy
// is missing. This is useful when transitioning an existing local deployment
// to cloud storage without downtime.
//
// Prerequisites:
//   - Valid AWS credentials (Access Key and Secret Key)
//   - Existing S3 bucket with appropriate permissions
//   - Network connectivity to AWS S3 services
package main

import (
	"context"
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

	awsBackend, err := aws.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create AWS backend:", err)
		return
	}

	if err := awsBackend.Ping(context.Background()); err != nil {
		fmt.Println("Failed to connect to AWS backend:", err)
		return
	}

	// Configure dual-write migration: local -> S3
	client, err := buckt.Default(
		buckt.WithLog(buckt.LogConfig{}),
		buckt.WithMigration(buckt.MigrationConfig{
			From: buckt.LocalBackend(),
			To:   awsBackend,
		}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	fmt.Println("Buckt Client initialized successfully (migration mode: local -> S3)")
}
