package main

import (
	"log"
	"mime/multipart"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/request"
)

func main() {
	b, err := buckt.NewBuckt("config.yaml", true, "/logs")
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Use the Buckt services directly

	var file *multipart.FileHeader = nil

	// Upload a file
	b.UploadFile(file, "mybucket", "hello/world")

	// Download a file
	data, err := b.DownloadFile(request.FileRequest{Filename: "hello.txt"})
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	log.Printf("Downloaded file: %s", string(data))

	// Delete a file
	err = b.DeleteFile(request.FileRequest{Filename: "hello.txt"})
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
	url, err := b.Serve(request.FileRequest{Filename: "hello.txt"}, true)
	if err != nil {
		log.Fatalf("Failed to serve file: %v", err)
	}
	log.Printf("Served file at: %s", url)
}
