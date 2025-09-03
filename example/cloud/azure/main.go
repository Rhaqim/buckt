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

	backend := buckt.RegisterPrimaryBackend(azure.NewBackend(cloudConfig))

	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{}), backend)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer client.Close()

	fmt.Println("Buckt Client initialized successfully")
}
