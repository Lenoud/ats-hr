// internal/shared/logger/logger.go
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Development bool
	Encoding    string // json or console
}

// Init initializes the global logger
func Init(cfg Config) error {
	var config zap.Config

	if cfg.Development {
		config = zap.NewDevelopmentConfig()
		config.Encoding = "console"
	} else {
		config = zap.NewProductionConfig()
		config.Encoding = cfg.Encoding
	}

	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return err
	}

	Log = logger.Sugar()
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	Log.Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(template string, args ...interface{}) {
	Log.Debugf(template, args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	Log.Info(args...)
}

// Infof logs a formatted info message
func Infof(template string, args ...interface{}) {
	Log.Infof(template, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	Log.Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(template string, args ...interface{}) {
	Log.Warnf(template, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	Log.Error(args...)
}

// Errorf logs a formatted error message
func Errorf(template string, args ...interface{}) {
	Log.Errorf(template, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	Log.Fatal(args...)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(template string, args ...interface{}) {
	Log.Fatalf(template, args...)
	os.Exit(1)
}

// With returns a logger with additional context fields
func With(args ...interface{}) *zap.SugaredLogger {
	return Log.With(args...)
}