package database

import (
	"database/sql"
	"time"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	*logger.BucktLogger
}

// NewSQLite creates a new SQLite database connection.
func NewDB(sqlDBInstance *sql.DB, driver string, log *logger.BucktLogger, debug bool) (*DB, error) {
	driverName := "sqlite"
	if driver == "postgres" || driver == "sqlite" {
		driverName = driver
	}

	// if debug is true, set log level to Info otherwise set to Silent
	var logLevel gormLogger.LogLevel = gormLogger.Silent
	if debug {
		logLevel = gormLogger.Info
	}

	// Create a new GORM configuration
	gormConfig := &gorm.Config{
		Logger: gormLogger.New(
			log.Logger,
			gormLogger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logLevel,
				Colorful:      true,
			},
		),
	}

	var dialector gorm.Dialector

	// Handle different database drivers
	switch driverName {
	case "postgres":
		if sqlDBInstance == nil {
			log.Warn("⚠️ No Postgres instance provided. Falling back to SQLite.")
			driverName = "sqlite"
		} else {
			dialector = postgres.New(postgres.Config{
				DriverName: driverName,
				Conn:       sqlDBInstance,
			})
		}
	}

	// Default to SQLite if necessary
	if driverName == "sqlite" {
		if sqlDBInstance != nil {
			dialector = sqlite.New(sqlite.Config{
				DriverName: driverName,
				Conn:       sqlDBInstance,
			})
		} else {
			log.Info("🛠️ Initializing new SQLite database (db.sqlite)...")
			dialector = sqlite.Open("db.sqlite")
		}
	}

	// Establish database connection
	log.Info("🚀 Connecting to " + driverName + " database...")
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, log.WrapError("Failed to connect to database", err)
	}

	// Access the underlying *sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		return nil, log.WrapError("Failed to get database connection:", err)
	}

	// Set connection pooling
	sqlDB.SetMaxOpenConns(10)                  // Max open connections
	sqlDB.SetMaxIdleConns(5)                   // Max idle connections
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Max connection lifetime

	// Optionally: Ping the database to ensure it's accessible
	if err := sqlDB.Ping(); err != nil {
		return nil, log.WrapError("Failed to ping database:", err)
	}

	return &DB{db, log}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return db.WrapError("Failed to get database connection: %v", err)
	}

	err = sqlDB.Close()

	return db.WrapError("Failed to close database connection: %v", err)
}

func (db *DB) Migrate() error {
	db.Info("🚀 Running migrations...")

	if err := db.AutoMigrate(&model.FolderModel{}); err != nil {
		return db.WrapErrorf("❌ failed to migrate FolderModel: %w", err)
	}
	db.Logger.Println("✅ FolderModel migrated")

	if err := db.AutoMigrate(&model.FileModel{}); err != nil {
		return db.WrapErrorf("❌ failed to migrate FileModel: %w", err)
	}
	db.Logger.Println("✅ FileModel migrated")

	return nil
}
