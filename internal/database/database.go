package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	log      domain.BucktLogger
	external bool
}

// NewSQLite creates a new SQLite database connection.
func NewDB(sqlDBInstance *sql.DB, driver model.DBDrivers, log domain.BucktLogger, silence bool) (*DB, error) {
	var external bool

	// Define supported database drivers
	supportedDrivers := map[model.DBDrivers]func(*sql.DB) gorm.Dialector{
		model.Postgres: func(db *sql.DB) gorm.Dialector {
			return postgres.New(postgres.Config{DriverName: "postgres", Conn: db})
		},
		model.SQLite: func(db *sql.DB) gorm.Dialector {
			return sqlite.New(sqlite.Config{DriverName: "sqlite", Conn: db})
		},
		// Add more drivers as needed:
		// "mysql": func(db *sql.DB) gorm.Dialector {
		//     return mysql.New(mysql.Config{DriverName: "mysql", Conn: db})
		// },
		// "mssql": func(db *sql.DB) gorm.Dialector {
		//     return sqlserver.New(sqlserver.Config{DriverName: "mssql", Conn: db})
		// },
	}

	driverString := string(driver)

	// If driver is empty or unsupported, fallback to SQLite
	if _, exists := supportedDrivers[driver]; !exists {
		log.Warn("⚠️ Unsupported or missing driver '" + driverString + "'. Falling back to SQLite.")
		driver = "sqlite"
	}

	// if silence is true, set log level to Info otherwise set to Silent
	var logLevel gormLogger.LogLevel = gormLogger.Silent
	if silence {
		logLevel = gormLogger.Info
	}

	// Create a new GORM configuration
	gormConfig := &gorm.Config{
		Logger: gormLogger.New(
			log.GetLogger(),
			gormLogger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logLevel,
				Colorful:      true,
			},
		),
	}

	// Determine the correct dialector
	var dialector gorm.Dialector
	if sqlDBInstance != nil {
		external = true
		dialector = supportedDrivers[driver](sqlDBInstance)
	} else {
		if driver == "sqlite" {
			log.Info("🛠️ Initializing new SQLite database (db.sqlite)...")
			dialector = sqlite.Open("db.sqlite")
		} else {
			return nil, log.WrapError("❌ No instance provided for '"+driverString+"' and cannot fall back to SQLite.", fmt.Errorf("no instance provided for '%s' ensure the database is running", driver))
		}
	}

	// Establish database connection
	log.Info("🚀 Connecting to " + driverString + " database...")
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, log.WrapError("Failed to connect to database", err)
	}

	if driver == "sqlite" {
		// enable foreign key support for sqlite
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return nil, log.WrapError("Failed to enable foreign key support for SQLite:", err)
		}
	}

	// Access the underlying *sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		return nil, log.WrapError("Failed to get database connection:", err)
	}

	// Set connection pooling
	if sqlDBInstance == nil {
		sqlDB.SetMaxOpenConns(10)                  // Max open connections
		sqlDB.SetMaxIdleConns(5)                   // Max idle connections
		sqlDB.SetConnMaxLifetime(30 * time.Minute) // Max connection lifetime
	}

	// Optionally: Ping the database to ensure it's accessible
	if err := sqlDB.Ping(); err != nil {
		return nil, log.WrapError("Failed to ping database:", err)
	}

	log.Info("🎉 Successfully connected to " + driverString + " database!")

	return &DB{db, log, external}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.external {
		db.log.Info("Skipping database close: external connection")
		return nil // Don't close external DB
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return db.log.WrapError("Failed to get database connection: %v", err)
	}

	err = sqlDB.Close()

	return err
}

func (db *DB) Migrate() error {
	db.log.Info("🚀 Running migrations...")

	if err := db.AutoMigrate(&model.FolderModel{}); err != nil {
		return db.log.WrapErrorf("❌ failed to migrate FolderModel: %w", err)
	}
	db.log.GetLogger().Println("✅ FolderModel migrated")

	if err := db.AutoMigrate(&model.FileModel{}); err != nil {
		return db.log.WrapErrorf("❌ failed to migrate FileModel: %w", err)
	}
	db.log.GetLogger().Println("✅ FileModel migrated")

	return nil
}
