package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web"
	"github.com/Rhaqim/buckt/pkg/events"
	"github.com/Rhaqim/buckt/pkg/metrics"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	withDB := false
	flatNamespaces := false

	// Allow overriding via command-line flags
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.BoolVar(&withDB, "db", withDB, "Use external database")
	flag.BoolVar(&flatNamespaces, "flat", flatNamespaces, "Use flat namespaces")
	flag.Parse()

	// Initialize the database
	var config buckt.ConfigFunc
	if withDB {
		db, err := InitDB()
		if err != nil {
			log.Fatalf("Failed to initialize the database: %v", err)
		}

		config = buckt.WithDB(buckt.Postgres, db)
	}

	// client is referenced by the upload handler below, so declare it first and
	// assign with '=' (not ':=') to keep the closure bound to this variable.
	var client *buckt.Client

	// onUpload fires after every successful upload. Here we generate image
	// derivatives and tag the file. Events run synchronously after the upload,
	// so in production you'd typically enqueue this to a worker instead of
	// resizing inline; for the demo, doing it here keeps things simple.
	onUpload := func(_ context.Context, e events.Event) {
		if e.Type != events.FileUploaded {
			return
		}
		if err := client.GenerateDerivatives(e.FileID); err != nil {
			log.Printf("derivative generation failed for %s: %v", e.FileID, err)
		}
		if err := client.SetFileMetadata(e.FileID, map[string]string{"source": "web-ui"}); err != nil {
			log.Printf("setting metadata failed for %s: %v", e.FileID, err)
		}
	}

	// Enable, in order: backend metrics (/metrics endpoint), content dedup,
	// image derivatives (thumbnail/medium), and the upload handler above.
	var err error
	client, err = buckt.Default(
		buckt.FlatNameSpaces(flatNamespaces),
		config,
		buckt.WithMetrics(metrics.NewCollector()),
		buckt.WithDedup(),
		buckt.WithImageDerivatives(
			buckt.DerivativeSpec{Name: "thumbnail", MaxWidth: 200},
			buckt.DerivativeSpec{Name: "medium", MaxWidth: 800},
		),
		buckt.WithEventHandler(onUpload),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer client.Close() // Ensure resources are cleaned up

	webClient, err := web.NewClient(client)
	if err != nil {
		log.Fatalf("Failed to create web client: %v", err)
	}

	// Start the router (optional, based on user choice)
	if err := webClient.Run(":" + *flagPort); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}

func InitDB() (*sql.DB, error) {
	var err error
	var db *sql.DB

	// Postgres database
	conn_string := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "password", "postgres")

	db, err = sql.Open("postgres", conn_string)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	return db, nil
}
