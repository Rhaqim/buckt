package main

import (
	"fmt"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/azure"
)

func main() {

	cloudConfig := azure.Config{
		AccountName: "accountName",
		AccountKey:  "accountKey",
		Container:   "container",
	}

	azureBackend, err := azure.NewBackend(cloudConfig)
	if err != nil {
		fmt.Println("Failed to create Azure backend:", err)
		return
	}

	backend := buckt.RegisterPrimaryBackend(azureBackend)

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
