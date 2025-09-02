package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/web"
)

func main() {
	// Initialize the database
	// db, err := InitDB()
	// if err != nil {
	// 	log.Fatalf("Failed to initialize the database: %v", err)
	// }
	b, err := buckt.Default(buckt.FlatNameSpaces(true))
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer b.Close() // Ensure resources are cleaned up

	// initialize router
	err = b.InitRouterService(buckt.WebModeAll)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt router: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	_, err = web.NewClient(b)
	if err != nil {
		log.Fatalf("Failed to create web client: %v", err)
	}

	// Allow overriding via command-line flag
	flagPort := flag.String("port", port, "Port to run the server on")
	flag.Parse()

	// Start the router (optional, based on user choice)
	if err := b.StartServer(":" + *flagPort); err != nil {
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
