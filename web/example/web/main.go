package main

import (
	"log"

	"github.com/Rhaqim/buckt"
	_ "github.com/Rhaqim/buckt/web"
)

func main() {
	opts := buckt.BucktConfig{
		Log: buckt.LogConfig{
			LogTerminal: true,
			Debug:       true,
		},
		MediaDir:       "media",
		FlatNameSpaces: true,
	}

	b, err := buckt.New(opts)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// initialize router
	err = b.InitRouterService(buckt.WebModeAll)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt router: %v", err)
	}

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":8080"); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
