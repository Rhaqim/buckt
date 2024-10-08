package main

import (
	"github.com/Rhaqim/buckt/config"
	"github.com/Rhaqim/buckt/internal/database"
	"github.com/Rhaqim/buckt/internal/web/router"
	"github.com/Rhaqim/buckt/pkg/logger"
)

func main() {
	// Load config
	cfg := config.LoadConfig("config.yaml")

	// Initialize logger
	log := logger.NewLogger(cfg.Log.LogToFileAndTerminal, cfg.Log.SaveDir)

	// Initialize database
	db, err := database.NewSQLite(cfg, log)
	if err != nil {
		log.ErrorLogger.Fatalf("Failed to initialize database: %v", err)
	}

	// Migrate the database
	db.Migrate()

	// Close the database connection when the main function exits
	defer db.Close()

	// Run the router
	router := router.NewRouter(log, cfg, db)
	router.Run()
}
