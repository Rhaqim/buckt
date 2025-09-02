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

	buckt, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}))
	if err != nil {
		fmt.Println(err)
		return
	}

	err = buckt.InitCloudService(cloudConfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer buckt.Close()

	fmt.Println("File uploaded successfully")
}
