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

func NewLogger(logToFileAndTerminal bool, saveDir string) *Logger {
	logDir := filepath.Join(saveDir, "logs", time.Now().Format("2006-01-02"))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("Failed to create log directory:", err)
	}

	// Open log files for today's date
	infoLogFile, err := os.OpenFile(filepath.Join(logDir, "info.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open info log file:", err)
	}

	errorLogFile, err := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open error log file:", err)
	}

	// Create multi-writer to write to both file and terminal if logToFileAndTerminal is true
	var infoWriter io.Writer = infoLogFile
	var errorWriter io.Writer = errorLogFile

	if logToFileAndTerminal {
		infoWriter = io.MultiWriter(os.Stdout, infoLogFile)
		errorWriter = io.MultiWriter(os.Stderr, errorLogFile)
	}

	// Initialize loggers
	infoLogger := log.New(infoWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger := log.New(errorWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	return &Logger{
		InfoLogger:  infoLogger,
		ErrorLogger: errorLogger,
	}
}

func (l *Logger) CleanLogs() {
	if err := os.RemoveAll("logs"); err != nil {
		log.Fatal("Failed to remove log directory:", err)
	}
}
