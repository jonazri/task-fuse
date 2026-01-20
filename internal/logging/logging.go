// Package logging provides structured logging infrastructure for task-fuse.
// It uses Go's standard log/slog package for structured logging with support
// for different log levels, formats (text/JSON), and output destinations.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LogLevel represents the logging verbosity level.
type LogLevel string

const (
	LevelError LogLevel = "error"
	LevelWarn  LogLevel = "warn"
	LevelInfo  LogLevel = "info"
	LevelDebug LogLevel = "debug"
)

// LogFormat represents the log output format.
type LogFormat string

const (
	FormatText LogFormat = "text"
	FormatJSON LogFormat = "json"
)

// Config holds the logging configuration.
type Config struct {
	Level  LogLevel
	Format LogFormat
	Output io.Writer // If nil, defaults to os.Stderr
}

// Logger wraps slog.Logger with additional convenience methods.
type Logger struct {
	*slog.Logger
	config Config
}

// DefaultConfig returns the default logging configuration.
func DefaultConfig() Config {
	return Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stderr,
	}
}

// ParseLevel parses a log level string and returns the corresponding LogLevel.
// Returns an error if the level string is invalid.
func ParseLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "error":
		return LevelError, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	default:
		return "", fmt.Errorf("invalid log level: %s (valid: error, warn, info, debug)", level)
	}
}

// ParseFormat parses a log format string and returns the corresponding LogFormat.
// Returns an error if the format string is invalid.
func ParseFormat(format string) (LogFormat, error) {
	switch strings.ToLower(format) {
	case "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid log format: %s (valid: text, json)", format)
	}
}

// toSlogLevel converts LogLevel to slog.Level.
func toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelError:
		return slog.LevelError
	case LevelWarn:
		return slog.LevelWarn
	case LevelInfo:
		return slog.LevelInfo
	case LevelDebug:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// NewLogger creates a new Logger with the given configuration.
func NewLogger(config Config) *Logger {
	output := config.Output
	if output == nil {
		output = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level: toSlogLevel(config.Level),
	}

	var handler slog.Handler
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		config: config,
	}
}

// NewLoggerFromFile creates a new Logger that writes to the specified file.
// If the file path is empty, it writes to stderr.
func NewLoggerFromFile(level LogLevel, format LogFormat, filePath string) (*Logger, *os.File, error) {
	config := Config{
		Level:  level,
		Format: format,
		Output: os.Stderr,
	}

	var file *os.File
	if filePath != "" {
		var err error
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		config.Output = file
	}

	return NewLogger(config), file, nil
}

// SetDefault sets the given logger as the default slog logger.
func SetDefault(logger *Logger) {
	slog.SetDefault(logger.Logger)
}

// Default returns a logger with default configuration.
func Default() *Logger {
	return NewLogger(DefaultConfig())
}

// WithComponent returns a new logger with the component field set.
// This is useful for identifying which component generated a log message.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
	}
}

// WithTaskID returns a new logger with the task ID field set.
func (l *Logger) WithTaskID(taskID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("task_id", taskID),
		config: l.config,
	}
}

// WithPath returns a new logger with the path field set.
func (l *Logger) WithPath(path string) *Logger {
	return &Logger{
		Logger: l.Logger.With("path", path),
		config: l.config,
	}
}
