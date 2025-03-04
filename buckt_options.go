package buckt

import (
	"database/sql"
	"log"

	"github.com/Rhaqim/buckt/internal/domain"
)

type Log struct {
	Logger      *log.Logger
	LogTerminal bool
	LoGfILE     string
	Debug       bool
}

// BucktOptions represents the configuration options for the Buckt application.
// It includes settings for logging, media directory, and standalone mode.
//
// Fields:
//
//	Log: Configuration for logging.
//	MediaDir: Path to the directory where media files are stored.
//	FlatNameSpaces: Flag indicating whether the application should use flat namespaces when storing files.
//	StandaloneMode: Flag indicating whether the application is running in standalone mode.
type BucktConfig struct {
	DB             *sql.DB
	Cache          domain.CacheManager
	Log            Log
	MediaDir       string
	FlatNameSpaces bool
	StandaloneMode bool
}
