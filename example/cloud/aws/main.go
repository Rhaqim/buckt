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

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), buckt.WithBackend(awsBackend))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
