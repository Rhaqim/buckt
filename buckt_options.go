package buckt

type Log struct {
	LogTerminal bool   `yaml:"logTerminal"`
	LoGfILE     string `yaml:"logFile"`
	Debug       bool   `yaml:"debug"`
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
type BucktOptions struct {
	Log            Log    `yaml:"log"`
	MediaDir       string `yaml:"mediaDir"`
	FlatNameSpaces bool   `yaml:"flatNameSpaces"`
	StandaloneMode bool   `yaml:"standaloneMode"`
}
