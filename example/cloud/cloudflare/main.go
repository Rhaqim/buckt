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

		Endpoint:     "https://custom-cloudflare-endpoint.com",
		UsePathStyle: true,
	}

	awsBackend, err := aws.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create AWS backend:", err)
		return
	}

	backend := buckt.RegisterPrimaryBackend(awsBackend)

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
