package main

import (
	"log"

	"github.com/Rhaqim/buckt"
)

func main() {
	b, err := buckt.Default(buckt.StandaloneMode(true), buckt.FlatNameSpaces(true))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":8080"); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
