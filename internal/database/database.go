package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/Rhaqim/buckt/internal/database/schema"
	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	gormschema "gorm.io/gorm/schema"
)

type DB struct {
	*gorm.DB
	log         domain.BucktLogger
	external    bool
	tablePrefix string
}

// NewDB creates a new database connection. tablePrefix is applied to every
// table name via gorm's NamingStrategy; the default (empty) prefix preserves
// buckt's historical table names (folder_models, file_models) so existing
// databases keep working unchanged.
func NewDB(sqlDBInstance *sql.DB, driver model.DBDrivers, log domain.BucktLogger, silence bool, tablePrefix string) (*DB, error) {
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
	logLevel := gormLogger.Silent
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
		// Apply the configurable table prefix to every model. An empty prefix
		// (the default) is equivalent to gorm's zero-value strategy, so existing
		// databases see the same table names as before.
		NamingStrategy: gormschema.NamingStrategy{TablePrefix: tablePrefix},
		// Translate dialect-specific driver errors (e.g. Postgres 23505, SQLite
		// "UNIQUE constraint failed") into gorm sentinels like ErrDuplicatedKey,
		// so the repository layer can map them to buckterr.ErrAlreadyExists
		// portably. Additive: First() still returns ErrRecordNotFound as before.
		TranslateError: true,
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

	return &DB{DB: db, log: log, external: external, tablePrefix: tablePrefix}, nil
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

// Migrate brings the database schema up to date using the versioned migration
// runner in internal/database/schema. Migrations are recorded in a ledger and
// each runs exactly once, so this is safe to call on every startup and adopts a
// pre-migration (v1.4.x) database in place without destroying data.
//
// The optional cloud file-migration feature table (MigrationModel) is created
// via a conditional AutoMigrate OUTSIDE the versioned ledger, preserving the
// v1.4.1 behavior where the table only exists when migration mode is enabled.
func (db *DB) Migrate(migrationEnabled bool) error {
	db.log.Info("🚀 Running migrations...")

	loader := schema.NewLoader(schema.DialectOf(db.DB), db.log, db.tablePrefix)
	if err := loader.Apply(context.Background(), db.DB); err != nil {
		return db.log.WrapErrorf("❌ schema migration failed", err)
	}

	if migrationEnabled {
		if err := db.AutoMigrate(&model.MigrationModel{}); err != nil {
			return db.log.WrapErrorf("❌ failed to migrate MigrationModel", err)
		}
		db.log.GetLogger().Println("✅ MigrationModel migrated")
	}

	return nil
}
