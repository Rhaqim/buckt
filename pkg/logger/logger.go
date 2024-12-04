package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
}

func NewLogger(logFile string, logTerminal bool) *Logger {
	// Default to logging to terminal if neither option is explicitly set
	if !logTerminal && logFile == "" {
		logTerminal = true
	}

	var infoWriter io.Writer = os.Stdout
	var errorWriter io.Writer = os.Stderr

	if logFile != "" {
		logDir := filepath.Join(logFile, "logs", time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatal("Failed to create log directory:", err)
		}

		infoLogFile, err := os.OpenFile(filepath.Join(logDir, "info.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Failed to open info log file:", err)
		}

		errorLogFile, err := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Failed to open error log file:", err)
		}

		// If logging to both file and terminal
		if logTerminal {
			infoWriter = io.MultiWriter(os.Stdout, infoLogFile)
			errorWriter = io.MultiWriter(os.Stderr, errorLogFile)
		} else {
			// Only log to file
			infoWriter = infoLogFile
			errorWriter = errorLogFile
		}
	}

	infoLogger := log.New(infoWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger := log.New(errorWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	return &Logger{
		InfoLogger:  infoLogger,
		ErrorLogger: errorLogger,
	}
}
