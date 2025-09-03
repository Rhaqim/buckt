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

	backend := buckt.RegisterPrimaryBackend(aws.NewBackend(cloudConfig))

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
