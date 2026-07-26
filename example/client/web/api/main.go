package main

import (
	"flag"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web"
	"github.com/Rhaqim/buckt/pkg/metrics"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Allow overriding via command-line flag
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.Parse()

	// Enable built-in backend metrics so the /metrics endpoint has data to show.
	client, err := buckt.Default(buckt.FlatNameSpaces(true), buckt.WithMetrics(metrics.NewCollector()))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer client.Close() // Ensure resources are cleaned up

	webClient, err := web.NewClient(client, web.Config{
		Mode: web.WebModeAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create web client: %v", err)
	}

	// Start the router (optional, based on user choice)
	if err := webClient.Run(":" + *flagPort); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}
