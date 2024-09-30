package database

import (
	"github.com/Rhaqim/buckt/config"
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
	var _DB *DB

	db, err := gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return nil, err
	}

	_DB = &DB{db, log}

	return _DB, nil
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
	if err := db.AutoMigrate(); err != nil {
		db.ErrorLogger.Fatalf("Failed to auto migrate database: %v", err)
	}
}
