package main

import (
	"fmt"
	"log"

	"github.com/Rhaqim/buckt"
)

func main() {
	opts := buckt.BucktConfig{
		Log: buckt.LogConfig{
			LogTerminal: true,
			LoGfILE:     "logs",
		},
		MediaDir: "media",
	}

	b, err := buckt.New(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Use the Buckt services directly

	//sample file
	var file []byte = []byte("sample file")

	var fileName string = "sample.txt"

	var constentType string = "application/octet-stream"

	// Upload a file
	id, err := b.UploadFile("user123", "", fileName, constentType, file)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}

	fmt.Println("File uploaded with ID:", id)

}
