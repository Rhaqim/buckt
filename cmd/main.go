package main

import (
	"flag"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
)

func main() {
	b, err := buckt.Default(buckt.StandaloneMode(true), buckt.FlatNameSpaces(true))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Allow overriding via command-line flag
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.Parse()

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":" + *flagPort); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
