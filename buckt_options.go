package buckt

import (
	"database/sql"
	"log"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/internal/model"
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

// CloudConfig stores configurations for different providers
// and their respective credentials.
//
// Fields:
//
//	Provider: The cloud provider to use.
//	Credentials: The credentials for the cloud provider.
type CloudConfig = model.CloudConfig

// AWSConfig credentials for AWS cloud provider.
//
// Fields:
//
//	AccessKey: The access key for AWS.
//	SecretKey: The secret key for AWS.
//	Region: The region for AWS.
//	Bucket: The bucket name for AWS.
type AWSConfig = model.AWSConfig

// AzureConfig credentials for Azure cloud provider.
//
// Fields:
//
//	AccountName: The account name for Azure.
//	AccountKey: The account key for Azure.
//	Container: The container name for Azure.
type AzureConfig = model.AzureConfig

// GCPConfig credentials for GCP cloud provider.
//
// Fields:
//
//	CredentialsFile: The path to the credentials file for GCP.
//	Bucket: The bucket name for GCP.
type GCPConfig = model.GCPConfig

type CloudProvider = model.CloudProvider

const (
	// CloudProviderNone represents no cloud provider.
	CloudProviderNone = model.CloudProviderNone

	// CloudProviderAWS represents the AWS cloud provider.
	CloudProviderAWS = model.CloudProviderAWS

	// CloudProviderAzure represents the Azure cloud provider.
	CloudProviderAzure = model.CloudProviderAzure

	// CloudProviderGCP represents the GCP cloud provider.
	CloudProviderGCP = model.CloudProviderGCP
)

type WebMode = model.WebMode

const (
	// WebModeAll registers all routes.
	WebModeAll = model.WebModeAll

	// WebModeAPI registers only the API routes.
	WebModeAPI = model.WebModeAPI

	// WebModeUI registers only the UI routes.
	WebModeUI = model.WebModeUI

	// WebModeMount registers only the API routes for the mount point.
	WebModeMount = model.WebModeMount
)

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
	Cloud          CloudConfig
}

type ConfigFunc func(*BucktConfig)

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

// WithCloud is a configuration function that sets the cloud provider and credentials
// for the BucktConfig. It takes a CloudConfig instance as an argument and assigns it
// to the Cloud field of BucktConfig.
//
// Parameters:
//   - cloud: An instance of CloudConfig to be used for cloud storage.
//
// Returns:
//   - A ConfigFunc that sets the Cloud field of BucktConfig.
func WithCloud(cloud CloudConfig) ConfigFunc {
	return func(c *BucktConfig) {
		c.Cloud = cloud
	}
}
