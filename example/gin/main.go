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

	// Start the router (optional, based on user choice)
	if err := b.Start(); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}