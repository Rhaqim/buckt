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

func (db *DB) Migrate(migrationEnabled bool) error {
	db.log.Info("🚀 Running migrations...")

	// Backward-compat: if upgrading from a version that used soft-deletes via
	// deleted_at, the column may still exist with rows that should remain hidden.
	// Hard-delete those rows so they don't suddenly reappear after we removed
	// the gorm.DeletedAt field from the model.
	if err := db.cleanupLegacySoftDeletes(); err != nil {
		db.log.Warn("legacy soft-delete cleanup skipped: " + err.Error())
	}

	if err := db.AutoMigrate(&model.FolderModel{}); err != nil {
		return db.log.WrapErrorf("❌ failed to migrate FolderModel: %w", err)
	}
	db.log.GetLogger().Println("✅ FolderModel migrated")

	if err := db.AutoMigrate(&model.FileModel{}); err != nil {
		return db.log.WrapErrorf("❌ failed to migrate FileModel: %w", err)
	}
	db.log.GetLogger().Println("✅ FileModel migrated")

	// Backfill file_models.user_id from the parent folder for any legacy rows
	// that were created before the user_id column existed.
	if err := db.backfillFileUserIDs(); err != nil {
		db.log.Warn("file user_id backfill skipped: " + err.Error())
	}

	if migrationEnabled {
		if err := db.AutoMigrate(&model.MigrationModel{}); err != nil {
			return db.log.WrapErrorf("❌ failed to migrate MigrationModel: %w", err)
		}
		db.log.GetLogger().Println("✅ MigrationModel migrated")
	}

	return nil
}

// backfillFileUserIDs populates file_models.user_id from the parent folder's
// user_id for any rows where user_id is empty. This handles legacy data from
// before the user_id column existed on file_models.
func (db *DB) backfillFileUserIDs() error {
	if !db.Migrator().HasColumn(&model.FileModel{}, "user_id") {
		return nil
	}

	// Count first so we only log when there's actually work to do
	var count int64
	if err := db.Model(&model.FileModel{}).Where("user_id = ? OR user_id IS NULL", "").Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}

	db.log.Infof("🔧 Backfilling user_id for %d legacy file rows...", count)

	// Update via subquery — works on both SQLite and Postgres
	err := db.Exec(`
		UPDATE file_models
		SET user_id = (
			SELECT folder_models.user_id
			FROM folder_models
			WHERE folder_models.id = file_models.parent_id
		)
		WHERE (user_id = '' OR user_id IS NULL)
		  AND EXISTS (
			SELECT 1 FROM folder_models WHERE folder_models.id = file_models.parent_id
		  )
	`).Error
	if err != nil {
		return err
	}

	db.log.GetLogger().Println("✅ Backfilled file user_id from parent folders")
	return nil
}

// cleanupLegacySoftDeletes hard-deletes any rows that were soft-deleted under
// the previous deleted_at scheme. Safe to call when the column doesn't exist
// (errors are ignored at the call site). This must run BEFORE AutoMigrate so
// that the deleted_at column hasn't been dropped yet.
func (db *DB) cleanupLegacySoftDeletes() error {
	hasFileCol := db.Migrator().HasColumn(&model.FileModel{}, "deleted_at")
	hasFolderCol := db.Migrator().HasColumn(&model.FolderModel{}, "deleted_at")

	if hasFileCol {
		if err := db.Exec("DELETE FROM file_models WHERE deleted_at IS NOT NULL").Error; err != nil {
			return err
		}
		db.log.GetLogger().Println("✅ Purged legacy soft-deleted files")
		// Drop the column so it doesn't linger
		if err := db.Migrator().DropColumn(&model.FileModel{}, "deleted_at"); err != nil {
			db.log.Warn("could not drop file_models.deleted_at column: " + err.Error())
		}
	}

	if hasFolderCol {
		if err := db.Exec("DELETE FROM folder_models WHERE deleted_at IS NOT NULL").Error; err != nil {
			return err
		}
		db.log.GetLogger().Println("✅ Purged legacy soft-deleted folders")
		if err := db.Migrator().DropColumn(&model.FolderModel{}, "deleted_at"); err != nil {
			db.log.Warn("could not drop folder_models.deleted_at column: " + err.Error())
		}
	}

	return nil
}
