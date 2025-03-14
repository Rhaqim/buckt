package main

import (
	"fmt"

	"github.com/Rhaqim/buckt"
)

func main() {
	// Create a new CloudConfig object
	cloudConfig := buckt.CloudConfig{
		Provider: buckt.CloudProviderAWS,
		Credentials: buckt.AWSConfig{
			AccessKey: "accessKey",
			SecretKey: "secretKey",
			Region:    "us-west-2",
			Bucket:    "my-bucket",
		},
	}

	buckt, err := buckt.Default(buckt.WithCloud(cloudConfig))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer buckt.Close()

	fmt.Println("File uploaded successfully")
}
