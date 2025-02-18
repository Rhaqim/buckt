package buckt

type Log struct {
	LogTerminal bool   `yaml:"logTerminal"`
	LoGfILE     string `yaml:"logFile"`
}

// BucktOptions represents the configuration options for the Buckt application.
// It includes settings for logging, media directory, and standalone mode.
//
// Fields:
//
//	Log: Configuration for logging.
//	MediaDir: Path to the directory where media files are stored.
//	StandaloneMode: Flag indicating whether the application is running in standalone mode.
type BucktOptions struct {
	Log            Log    `yaml:"log"`
	MediaDir       string `yaml:"mediaDir"`
	StandaloneMode bool   `yaml:"standaloneMode"`
}
