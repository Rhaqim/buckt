package database

import (
	"time"

	"github.com/Rhaqim/buckt/config"
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
func NewSQLite(cfg *config.Config, log *logger.Logger) (*DB, error) {

	db, err := gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Access the underlying *sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pooling
	sqlDB.SetMaxOpenConns(10)                  // Max open connections
	sqlDB.SetMaxIdleConns(5)                   // Max idle connections
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Max connection lifetime

	// Optionally: Ping the database to ensure it's accessible
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return &DB{db, log}, nil
}

// Close closes the database connection.
func (db *DB) Close() {
	sqlDB, err := db.DB.DB()
	if err != nil {
		db.ErrorLogger.Fatalf("Failed to get database connection: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		db.ErrorLogger.Fatalf("Failed to close database connection: %v", err)
	}
}

func (db *DB) Migrate() {
	if err := db.AutoMigrate(&model.FileModel{}, &model.BucketModel{}, &model.OwnerModel{}, &model.TagModel{}, &model.FolderModel{}); err != nil {
		db.ErrorLogger.Fatalf("Failed to auto migrate database: %v", err)
	}
}
