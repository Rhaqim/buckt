package main

import (
	"log"

	"github.com/Rhaqim/buckt"
)

func main() {
	opts := buckt.BucktConfig{
		Log: buckt.Log{
			LogTerminal: true,
			Debug:       true,
		},
		MediaDir:       "media",
		StandaloneMode: true,
		FlatNameSpaces: true,
	}

	b, err := buckt.New(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":8080"); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
