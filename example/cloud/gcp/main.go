package main

import (
	"fmt"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/gcp"
)

func main() {

	cloudConfig := gcp.Config{
		CredentialsFile: "path/to/credentials.json",
		Bucket:          "my-bucket",
	}

	gcpBackend, err := gcp.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create GCP backend:", err)
		return
	}

	backend := buckt.RegisterPrimaryBackend(gcpBackend)

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
