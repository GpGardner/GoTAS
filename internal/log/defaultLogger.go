package logger

import (
	"log"
	"os"
)

type DefaultLogger struct {
	*log.Logger
	level LogLevel
}

// validate that DefaultLogger implements the Logger interface
var _ Logger = (*DefaultLogger)(nil)

// NewDefaultLogger creates a new instance of DefaultLogger.
// This logger uses the standard log package to log messages.
// It is a simple implementation of the Logger interface.
// It can be used for basic logging needs without any additional configuration.
// This logger is not thread-safe and should be used in a single-threaded context.
// If you need a thread-safe logger, consider using a more advanced logging library.
// Example usage:
// logger := logger.NewDefaultLogger()
// logger.Info("This is an info message")
// logger.Error("This is an error message", err)
// logger.Debug("This is a debug message")
// logger.Warn("This is a warning message")
// logger.Fatal("This is a fatal message")
func NewDefaultLogger(lvl LogLevel) *DefaultLogger {
	return &DefaultLogger{
		Logger: log.New(os.Stdout, "", log.LstdFlags), // Write logs to stdout
		level:  lvl,
	}
}

func (l DefaultLogger) Info(msg string, args ...any) {
	if l.level <= LogLevelInfo {
		log.Printf("[INFO] "+msg, args...)
	}
}

func (l DefaultLogger) Error(msg string, args ...any) {
	if l.level <= LogLevelError {
		log.Printf("[ERROR] "+msg, args...)
	}
}

func (l DefaultLogger) Debug(msg string, args ...any) {
	if l.level <= LogLevelDebug {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l DefaultLogger) Warn(msg string, args ...any) {
	if l.level <= LogLevelWarn {
		log.Printf("[WARN] "+msg, args...)
	}
}

func (l DefaultLogger) Fatal(msg string, args ...any) {
	if l.level <= LogLevelFatal {
		log.Fatalf("[FATAL] "+msg, args...)
	}
}
