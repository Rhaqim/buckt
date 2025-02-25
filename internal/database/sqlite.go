package database

import (
	"time"

	"github.com/Rhaqim/buckt/internal/model"
	"github.com/Rhaqim/buckt/pkg/logger"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	*logger.Logger
}

// NewSQLite creates a new SQLite database connection.
func NewSQLite(log *logger.Logger, debug bool) (*DB, error) {
	// if debug is true, set log level to Info otherwise set to Silent
	var logLevel gormLogger.LogLevel
	if debug {
		logLevel = gormLogger.Info
	} else {
		logLevel = gormLogger.Silent
	}

	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{
		Logger: gormLogger.New(
			log.InfoLogger,
			gormLogger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logLevel,
				Colorful:      true,
			},
		),
	})

	if err != nil {
		return nil, log.WrapError("Failed to connect to database:", err)
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
	err := db.AutoMigrate(&model.FileModel{}, &model.FolderModel{})
	if err != nil {
		return db.WrapError("Failed to auto migrate database:", err)
	}

	return nil
}
