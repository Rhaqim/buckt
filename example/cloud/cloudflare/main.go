// Package main demonstrates using Cloudflare R2 as a storage backend via the
// S3-compatible cloud/aws module.
//
// R2 endpoints have the form: https://<ACCOUNT_ID>.r2.cloudflarestorage.com
// The cloud/aws backend auto-detects R2 from the endpoint suffix and enables
// path-style addressing automatically.
//
// Get your credentials from: https://dash.cloudflare.com/?to=/:account/r2/api-tokens
package main

import (
	"context"
	"fmt"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/aws"
)

func main() {

	cloudConfig := aws.Config{
		AccessKey: "your-r2-access-key-id",
		SecretKey: "your-r2-secret-access-key",
		Bucket:    "my-bucket",

		// R2 endpoint format: https://<ACCOUNT_ID>.r2.cloudflarestorage.com
		// The aws backend detects R2 from this suffix and configures path-style addressing.
		Endpoint: "https://<ACCOUNT_ID>.r2.cloudflarestorage.com",

		// Region: leave empty for R2 — the backend defaults to "auto".
		// Region: "",

		// UsePathStyle is auto-enabled for R2 endpoints; only set this true
		// for other S3-compatible services like MinIO.
		// UsePathStyle: false,
	}

	r2Backend, err := aws.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create Cloudflare R2 backend:", err)
		return
	}

	if err := r2Backend.Ping(context.Background()); err != nil {
		fmt.Println("Failed to connect to Cloudflare R2:", err)
		return
	}

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), buckt.WithBackend(r2Backend))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully with Cloudflare R2 backend")
}
