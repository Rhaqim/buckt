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
func NewSQLite(log *logger.Logger) (*DB, error) {

	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
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
	return db.WrapError("Failed to auto migrate database:", err)
}
