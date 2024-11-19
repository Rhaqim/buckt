package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Database struct {
	DSN string `yaml:"dsn"`
}

type Server struct {
	Port string `yaml:"port"`
}

type Media struct {
	Dir string `yaml:"dir"`
}

type Log struct {
	Level                string `yaml:"level"`
	LogToFileAndTerminal bool   `yaml:"logToFileAndTerminal"`
	SaveDir              string `yaml:"saveDir"`
}

type Endpoint struct {
	URL string `yaml:"url"`
}

type Config struct {
	Log       Log      `yaml:"log"`
	Database  Database `yaml:"database"`
	Server    Server   `yaml:"server"`
	Media     Media    `yaml:"media"`
	Endpoint  Endpoint `yaml:"endpoint"`
	Templates string   `yaml:"templates"`
}

// LoadConfig loads the configuration from the given file.
func LoadConfig(configPath string) *Config {
	var cfg = &Config{}

	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := yaml.Unmarshal(file, cfg); err != nil {
		log.Fatal(err)
	}

	// convert Server.Port to ":port"
	cfg.Server.Port = ":" + cfg.Server.Port

	return cfg
}
