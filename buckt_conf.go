package buckt

import (
	"database/sql"
	"log"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
)

type DBDrivers = model.DBDrivers // Type alias

const (
	// Posstgres represents the Postgres database driver.
	Postgres = model.Postgres
	// SQLite represents the SQLite database driver.
	SQLite = model.SQLite
)

// DBConfig holds the configuration for the database connection.
//
// Fields:
//
//	Driver: A string representing the database driver name.
//	Database: A pointer to an sql.DB instance representing the database connection.
type DBConfig struct {
	Driver   DBDrivers
	Database *sql.DB
}

// FileCacheConfig holds the configuration for the file cache.
// It includes settings for the number of counters, maximum cost,
// and buffer items.
//
// Fields:
//
//	NumCounters: The number of counters for the cache.
//	MaxCost: The maximum cost of the cache.
//	BufferItems: The number of items in the buffer.
type FileCacheConfig struct {
	NumCounters int64
	MaxCost     int64
	BufferItems int64
}

// Validate checks the configuration values for the file cache.
// It sets default values if the provided values are less than or equal to zero.
// The default values are:
//
//	NumCounters: 1e7 (10 million)
//	MaxCost: 1 << 30 (1 GB)
//	BufferItems: 64
//
// If the values are already set to valid values, they remain unchanged.
// This function is useful for ensuring that the cache configuration is valid
// before using it in the application.
// It is typically called during the initialization of the cache manager.
func (f *FileCacheConfig) Validate() {
	if f.NumCounters <= 0 {
		f.NumCounters = 1e7 // 10M
	}
	if f.MaxCost <= 0 {
		f.MaxCost = 1 << 30 // 1GB
	}
	if f.BufferItems <= 0 {
		f.BufferItems = 64
	}
}

// CacheConfig holds the configuration for the cache manager.
// It includes the cache manager instance and file cache configuration.
// Fields:
//
//	Manager: The cache manager instance.
//	FileCacheConfig: The file cache configuration.
type CacheConfig struct {
	Manager domain.CacheManager
	FileCacheConfig
}

// LogConfig holds the configuration for logging in the application.
//
// Fields:
//
//	LogFile: A string representing the log file path.
//	Debug: A boolean flag indicating whether to enable debug mode.
//	LogTerminal: A boolean flag indicating whether to log to the terminal.
//	Logger: A pointer to a log.Logger instance. If nil, a new logger will be created.
type LogConfig struct {
	LogFile     string
	Debug       bool
	LogTerminal bool
	Logger      *log.Logger
}

// Backend represents the file backend interface.
//
// It defines methods for interacting with the file storage backend.
type Backend = domain.FileBackend

// FileInfo represents information about a file.
type FileInfo = model.FileInfo

// BackendConfig holds the configuration for the file backend.
//
// It includes the source and target backends for migration.
//
// If only one backend is specified, it is used as the primary backend.
type BackendConfig struct {
	// Source is the current backend in use (e.g., local, S3).
	Source Backend

	// Target is the backend to migrate to (e.g., S3, Azure).
	// If MigrationEnabled is false, this is ignored.
	Target Backend

	// MigrationEnabled enables dual-write migration mode.
	MigrationEnabled bool
}

// LocalBackend is a placeholder and is replaced with the actual local backend implementation.
//
// Usecase: Selecting the local backend for migration.
//
//	backendConfig := BackendConfig{
//	 MigrationEnabled: true,
//		Source: buckt.LocalBackend(),
//		Target: aws.S3Backend(),
//	}
func LocalBackend() Backend {
	return &domain.PlaceholderBackend{Title: "local"}
}

// BucktOptions represents the configuration options for the Buckt application.
// It includes settings for logging, media directory, and standalone mode.
//
// Fields:
//
//	DB: Database configuration.
//	Cache: Cache configuration.
//	Log: Configuration for logging.
//	MediaDir: Path to the directory where media files are stored.
//	FlatNameSpaces: Flag indicating whether the application should use flat namespaces when storing files.
type Config struct {
	MediaDir       string
	FlatNameSpaces bool

	DB      DBConfig
	Cache   CacheConfig
	Log     LogConfig
	Backend BackendConfig
}

type ConfigFunc func(*Config)

// WithDB is a configuration function that sets the database connection
// for the Config. It takes a *sql.DB as an argument and returns
// a ConfigFunc that assigns the provided database connection to the
// Config.
//
// Parameters:
//   - driver: A string representing the database driver name.
//   - db: A pointer to an sql.DB instance representing the database connection.
//
// Returns:
//   - ConfigFunc: A function that takes a pointer to Config and sets its DB field.
func WithDB(driver DBDrivers, db *sql.DB) ConfigFunc {
	return func(c *Config) {
		c.DB.Driver = DBDrivers(driver)
		c.DB.Database = db
	}
}

// WithLogger is a configuration function that sets the logger for the Config.
// It takes a Log instance as an argument and assigns it to the Log field of Config.
//
// Parameters:
//   - log: An instance of Log to be used for logging.
//
// Returns:
//   - A ConfigFunc that sets the Log field of Config.
//     A ConfigFunc that sets the CacheManager in the Config.
func WithCache(cache CacheConfig) ConfigFunc {
	return func(c *Config) {
		c.Cache = cache
	}
}

// WithLog is a configuration function that sets the logger for the Config.
// It takes a Log instance as an argument and assigns it to the Log field of Config.
//
// Parameters:
//   - log: An instance of Log to be used for logging.
//
// Returns:
//   - A ConfigFunc that sets the Log field of Config.
func WithLog(log LogConfig) ConfigFunc {
	return func(c *Config) {
		c.Log = log
	}
}

// MediaDir sets the directory path for media files in the Config.
// It takes a string parameter mediaDir which specifies the path to the media directory.
// It returns a ConfigFunc that updates the MediaDir field of Config.
func MediaDir(mediaDir string) ConfigFunc {
	return func(c *Config) {
		c.MediaDir = mediaDir
	}
}

// FlatNameSpaces is a configuration function that sets the FlatNameSpaces
// option in the Config. When the flat parameter is true, it enables
// flat namespaces; otherwise, it disables them.
//
// Parameters:
//
//	flat - a boolean value indicating whether to enable or disable flat namespaces.
//
// Returns:
//
//	A ConfigFunc that applies the flat namespaces setting to a Config.
func FlatNameSpaces(flat bool) ConfigFunc {
	return func(c *Config) {
		c.FlatNameSpaces = flat
	}
}

// RegisterPrimaryBackend registers the primary backend for the Buckt application.
func RegisterPrimaryBackend(backend Backend) ConfigFunc {
	return func(c *Config) {
		c.Backend.Source = backend
	}
}

// RegisterSecondaryBackend registers the secondary backend for the Buckt application.
func RegisterSecondaryBackend(backend Backend) ConfigFunc {
	return func(c *Config) {
		c.Backend.Target = backend
	}
}
