package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type BucktLogger struct {
	Logger *log.Logger
	debug  bool
}

type LogFunc func(*BucktLogger)

func NewLogger(logFile string, logTerminal, debug bool, opts ...LogFunc) *BucktLogger {
	bucktLogger := &BucktLogger{debug: debug}

	for _, opt := range opts {
		opt(bucktLogger)
	}

	if !logTerminal && logFile == "" {
		logTerminal = true
	}

	var logWriter io.Writer

	// Setup log file if provided
	if logFile != "" {
		logDir := filepath.Join(logFile, time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatal("Failed to create log directory:", err)
		}

		infoLogFile, err := os.OpenFile(filepath.Join(logDir, "info.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Failed to open info log file:", err)
		}

		if logTerminal && !debug {
			logWriter = io.MultiWriter(os.Stdout, infoLogFile)
		} else {
			logWriter = infoLogFile
		}
	} else {
		// If no log file and debugging, silence logs; otherwise, use stdout
		if logTerminal && !debug {
			logWriter = io.Discard // Silence terminal logs
		} else {
			logWriter = os.Stdout
		}
	}

	if bucktLogger.Logger == nil {
		bucktLogger.Logger = log.New(logWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	return bucktLogger
}

func WithLogger(logger *log.Logger) LogFunc {
	return func(l *BucktLogger) {
		l.Logger = logger
	}
}

// Writer returns the writer for the info logger
func (l *BucktLogger) Writer() io.Writer {
	return l.Logger.Writer()
}

// Info logs an info message
func (l *BucktLogger) Info(message string) {
	if !l.debug {
		l.Logger.Println(message)
	}
}

// Warn logs a warning message
func (l *BucktLogger) Warn(message string) {
	if !l.debug {
		l.Logger.Println("WARN:", message)
	}
}

// Error logs an error message
func (l *BucktLogger) Errorf(format string, args ...any) {
	if !l.debug {
		l.Logger.Printf("ERROR: "+format, args...)
	}
}

// WrapError logs an error message and returns an error
func (l *BucktLogger) WrapError(message string, err error) error {
	if !l.debug {
		l.Logger.Println("ERROR:", message, err)
	}
	return err
}

// WrapErrorf logs an error message with formatting
func (l *BucktLogger) WrapErrorf(message string, err error, args ...any) error {
	if !l.debug {
		l.Logger.Printf("ERROR: %s %v\n", message+" "+err.Error(), args)
	}
	return err
}
