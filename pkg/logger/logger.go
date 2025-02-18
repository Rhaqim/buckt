package logger

import (
	"errors"
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

		if logTerminal {
			infoWriter = io.MultiWriter(os.Stdout, infoLogFile)
			errorWriter = io.MultiWriter(os.Stderr, errorLogFile)
		} else {
			infoWriter = infoLogFile
			errorWriter = errorLogFile
		}
	}

	return &Logger{
		InfoLogger:  log.New(infoWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		ErrorLogger: log.New(errorWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Writer returns the writer for the info logger
func (l *Logger) Writer() io.Writer {
	return l.InfoLogger.Writer()
}

// Success logs a success message
func (l *Logger) Success(message string) {
	l.InfoLogger.Println(message)
}

// Error logs an error message and returns an error type
func (l *Logger) Error(userMsg, devMsg string) error {
	err := errors.New(devMsg)
	l.ErrorLogger.Printf("%s | Details: %s", userMsg, devMsg)
	return err
}

// WrapError logs an error and returns a wrapped error type
func (l *Logger) WrapError(userMsg string, err error) error {
	if err == nil {
		l.Success("Success: " + userMsg)
		return nil
	}
	l.ErrorLogger.Printf("%s | Error: %s", userMsg, err.Error())
	return errors.New(userMsg + ": " + err.Error())
}
