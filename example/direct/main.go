package main

import (
	"log"

	"github.com/Rhaqim/buckt"
)

func main() {
	b, err := buckt.NewBuckt("config.yaml", true, "/logs")
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Use the Buckt services directly

	// Upload a file
	b.UploadFile([]byte("Hello, World!"), "mybucket", "hello.txt")

	// Download a file
	data, err := b.DownloadFile("hello.txt")
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	log.Printf("Downloaded file: %s", string(data))

	// Delete a file
	err = b.DeleteFile("hello.txt")
	if err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}

	// Create a bucket
	err = b.CreateBucket("mybucket", "My bucket", "ownerID")
	if err != nil {
		log.Fatalf("Failed to create bucket: %v", err)
	}

	// Create an owner
	err = b.CreateOwner("owner", "owner@gmail.com")
	if err != nil {
		log.Fatalf("Failed to create owner: %v", err)
	}

	// Serve a file
	url, err := b.Serve("hello.txt", true)
	if err != nil {
		log.Fatalf("Failed to serve file: %v", err)
	}
	log.Printf("Served file at: %s", url)
}
