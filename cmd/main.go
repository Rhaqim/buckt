package main

import (
	"log"

	"github.com/Rhaqim/buckt"
)

func main() {
	opts := buckt.BucktOptions{
		Log: buckt.Log{
			Level:       "debug",
			LogTerminal: true,
			LoGfILE:     "buckt.log",
		},
		MediaDir:       "media",
		StandaloneMode: true,
	}

	b, err := buckt.NewBuckt(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":8080"); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
