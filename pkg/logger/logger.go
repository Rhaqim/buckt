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
}

type LogFunc func(*BucktLogger)

func NewLogger(logFile string, logTerminal bool, opts ...LogFunc) *BucktLogger {
	bucktLogger := &BucktLogger{}

	for _, opt := range opts {
		opt(bucktLogger)
	}

	if !logTerminal && logFile == "" {
		logTerminal = true
	}

	var logWriter io.Writer = os.Stdout

	if logFile != "" {
		logDir := filepath.Join(logFile, time.Now().Format("2006-01-02"))
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatal("Failed to create log directory:", err)
		}

		infoLogFile, err := os.OpenFile(filepath.Join(logDir, "info.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Failed to open info log file:", err)
		}

		if logTerminal {
			logWriter = io.MultiWriter(os.Stdout, infoLogFile)
		} else {
			logWriter = infoLogFile
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

// Success logs a success message
func (l *BucktLogger) Info(message string) {
	l.Logger.Println(message)
}

// Error logs an error message and returns an error type
func (l *BucktLogger) Errorf(format string, args ...interface{}) {

	message := format
	if len(args) > 0 {
		message = format
	}
	l.Logger.Println("ERROR:", message)
}

func (l *BucktLogger) WrapError(message string, err error) error {
	l.Logger.Println("ERROR:", message, err)
	return err
}

func (l *BucktLogger) WrapErrorf(message string, err error, args ...interface{}) error {
	l.Logger.Printf("ERROR: %s %v\n", message+" "+err.Error(), args)
	return err
}
