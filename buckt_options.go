package buckt

type Log struct {
	// Show logs in terminal
	LogTerminal bool `yaml:"logTerminal"`

	// Save logs to a file e.g buckt.log
	LoGfILE string `yaml:"logFile"`
}

// BucktOptions is a struct that holds the options for the Buckt service
type BucktOptions struct {
	// Log options
	Log Log `yaml:"log"`

	// Media directory to store media files
	MediaDir string `yaml:"mediaDir"`

	// Run as standalone server or as a part of an existing server
	StandaloneMode bool `yaml:"standaloneMode"`
}
