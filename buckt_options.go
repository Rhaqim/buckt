package buckt

type Log struct {
	Level       string `yaml:"level"`
	LogTerminal bool   `yaml:"logTerminal"`
	LoGfILE     string `yaml:"logFile"`
}

type BucktOptions struct {
	Log            Log    `yaml:"log"`
	MediaDir       string `yaml:"mediaDir"`
	StandaloneMode bool   `yaml:"standaloneMode"`
}
