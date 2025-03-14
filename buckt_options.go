package buckt

import (
	"database/sql"
	"log"

	"github.com/Rhaqim/buckt/internal/domain"
)

// LogConfig holds the configuration for logging in the application.
//
// Fields:
//
//	Logger: A pointer to a log.Logger instance. If nil, a new logger will be created.
//	LogTerminal: A boolean flag indicating whether to log to the terminal.
//	LogFile: A string representing the log file path.
//	Debug: A boolean flag indicating whether to enable debug mode.
type LogConfig struct {
	Logger      *log.Logger
	LogTerminal bool
	LogFile     string
	Debug       bool
}

type DBDrivers = domain.DBDrivers // Type alias

const (
	Postgres = domain.Postgres
	SQLite   = domain.SQLite
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

// BucktOptions represents the configuration options for the Buckt application.
// It includes settings for logging, media directory, and standalone mode.
//
// Fields:
//
//	DB: Database configuration.
//	Cache: CacheManager instance.
//	Log: Configuration for logging.
//	MediaDir: Path to the directory where media files are stored.
//	FlatNameSpaces: Flag indicating whether the application should use flat namespaces when storing files.
//	StandaloneMode: Flag indicating whether the application is running in standalone mode.
type BucktConfig struct {
	DB             DBConfig
	Cache          domain.CacheManager
	Log            LogConfig
	MediaDir       string
	FlatNameSpaces bool
	StandaloneMode bool
}

type ConfigFunc func(*BucktConfig)

// StandaloneMode sets the standalone mode for the BucktConfig.
// When standalone mode is enabled, the application will run independently
// without relying on external services or configurations.
//
// Parameters:
//
//	standalone - a boolean value indicating whether to enable standalone mode.
//
// Returns:
//
//	A ConfigFunc that sets the standalone mode in the BucktConfig.
func StandaloneMode(standalone bool) ConfigFunc {
	return func(c *BucktConfig) {
		c.StandaloneMode = standalone
	}
}

// FlatNameSpaces is a configuration function that sets the FlatNameSpaces
// option in the BucktConfig. When the flat parameter is true, it enables
// flat namespaces; otherwise, it disables them.
//
// Parameters:
//
//	flat - a boolean value indicating whether to enable or disable flat namespaces.
//
// Returns:
//
//	A ConfigFunc that applies the flat namespaces setting to a BucktConfig.
func FlatNameSpaces(flat bool) ConfigFunc {
	return func(c *BucktConfig) {
		c.FlatNameSpaces = flat
	}
}

// MediaDir sets the directory path for media files in the BucktConfig.
// It takes a string parameter mediaDir which specifies the path to the media directory.
// It returns a ConfigFunc that updates the MediaDir field of BucktConfig.
func MediaDir(mediaDir string) ConfigFunc {
	return func(c *BucktConfig) {
		c.MediaDir = mediaDir
	}
}

// WithLogger is a configuration function that sets the logger for the BucktConfig.
// It takes a Log instance as an argument and assigns it to the Log field of BucktConfig.
//
// Parameters:
//   - log: An instance of Log to be used for logging.
//
// Returns:
//   - A ConfigFunc that sets the Log field of BucktConfig.
//     A ConfigFunc that sets the CacheManager in the BucktConfig.
func WithCache(cache domain.CacheManager) ConfigFunc {
	return func(c *BucktConfig) {
		c.Cache = cache
	}
}

// WithDB is a configuration function that sets the database connection
// for the BucktConfig. It takes a *sql.DB as an argument and returns
// a ConfigFunc that assigns the provided database connection to the
// BucktConfig.
//
// Parameters:
//   - driver: A string representing the database driver name.
//   - db: A pointer to an sql.DB instance representing the database connection.
//
// Returns:
//   - ConfigFunc: A function that takes a pointer to BucktConfig and sets its DB field.
func WithDB(driver DBDrivers, db *sql.DB) ConfigFunc {
	return func(c *BucktConfig) {
		c.DB.Driver = DBDrivers(driver)
		c.DB.Database = db
	}
}

// WithLog is a configuration function that sets the logger for the BucktConfig.
// It takes a Log instance as an argument and assigns it to the Log field of BucktConfig.
//
// Parameters:
//   - log: An instance of Log to be used for logging.
//
// Returns:
//   - A ConfigFunc that sets the Log field of BucktConfig.
func WithLog(log LogConfig) ConfigFunc {
	return func(c *BucktConfig) {
		c.Log = log
	}
}
