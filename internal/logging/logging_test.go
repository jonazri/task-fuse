package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
		wantErr  bool
	}{
		{"error", LevelError, false},
		{"ERROR", LevelError, false},
		{"Error", LevelError, false},
		{"warn", LevelWarn, false},
		{"WARN", LevelWarn, false},
		{"warning", LevelWarn, false},
		{"WARNING", LevelWarn, false},
		{"info", LevelInfo, false},
		{"INFO", LevelInfo, false},
		{"debug", LevelDebug, false},
		{"DEBUG", LevelDebug, false},
		{"invalid", "", true},
		{"", "", true},
		{"trace", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected LogFormat
		wantErr  bool
	}{
		{"text", FormatText, false},
		{"TEXT", FormatText, false},
		{"Text", FormatText, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"Json", FormatJSON, false},
		{"invalid", "", true},
		{"", "", true},
		{"xml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name: "error level text format",
			config: Config{
				Level:  LevelError,
				Format: FormatText,
				Output: os.Stderr,
			},
		},
		{
			name: "debug level json format",
			config: Config{
				Level:  LevelDebug,
				Format: FormatJSON,
				Output: os.Stderr,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.config)
			if logger == nil {
				t.Error("NewLogger() returned nil")
			}
			if logger.Logger == nil {
				t.Error("NewLogger().Logger is nil")
			}
		})
	}
}

func TestLoggerTextOutput(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: &buf,
	}

	logger := NewLogger(config)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Log output should contain 'key=value', got: %s", output)
	}
}

func TestLoggerJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: FormatJSON,
		Output: &buf,
	}

	logger := NewLogger(config)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("JSON log output should contain '\"msg\":\"test message\"', got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("JSON log output should contain '\"key\":\"value\"', got: %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelWarn,
		Format: FormatText,
		Output: &buf,
	}

	logger := NewLogger(config)

	// Debug and Info should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	if strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message should be filtered at Warn level")
	}
	if strings.Contains(buf.String(), "info message") {
		t.Error("Info message should be filtered at Warn level")
	}

	// Warn and Error should pass through
	logger.Warn("warn message")
	logger.Error("error message")

	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message should not be filtered at Warn level")
	}
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message should not be filtered at Warn level")
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: &buf,
	}

	logger := NewLogger(config)
	componentLogger := logger.WithComponent("test-component")
	componentLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component=test-component") {
		t.Errorf("Log output should contain 'component=test-component', got: %s", output)
	}
}

func TestWithTaskID(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: &buf,
	}

	logger := NewLogger(config)
	taskLogger := logger.WithTaskID("1.2.3")
	taskLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "task_id=1.2.3") {
		t.Errorf("Log output should contain 'task_id=1.2.3', got: %s", output)
	}
}

func TestWithPath(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: &buf,
	}

	logger := NewLogger(config)
	pathLogger := logger.WithPath("/pending/1.task.md")
	pathLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "path=/pending/1.task.md") {
		t.Errorf("Log output should contain 'path=/pending/1.task.md', got: %s", output)
	}
}

func TestNewLoggerFromFile(t *testing.T) {
	// Test with empty file path (should use stderr)
	logger, file, err := NewLoggerFromFile(LevelInfo, FormatText, "")
	if err != nil {
		t.Errorf("NewLoggerFromFile with empty path should not error: %v", err)
	}
	if file != nil {
		t.Error("NewLoggerFromFile with empty path should return nil file handle")
	}
	if logger == nil {
		t.Error("NewLoggerFromFile should return a logger")
	}

	// Test with actual file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, file, err = NewLoggerFromFile(LevelDebug, FormatText, logPath)
	if err != nil {
		t.Errorf("NewLoggerFromFile should not error: %v", err)
	}
	if file == nil {
		t.Error("NewLoggerFromFile should return file handle")
	}
	if logger == nil {
		t.Error("NewLoggerFromFile should return a logger")
	}

	// Write a log message
	logger.Info("test log message")

	// Close the file
	file.Close()

	// Verify the file contains the log message
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "test log message") {
		t.Errorf("Log file should contain 'test log message', got: %s", string(content))
	}
}

func TestNewLoggerFromFileError(t *testing.T) {
	// Test with invalid file path
	_, _, err := NewLoggerFromFile(LevelInfo, FormatText, "/nonexistent/directory/test.log")
	if err == nil {
		t.Error("NewLoggerFromFile should error with invalid path")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Default() should return a logger")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.Level != LevelInfo {
		t.Errorf("DefaultConfig().Level = %v, want %v", config.Level, LevelInfo)
	}
	if config.Format != FormatText {
		t.Errorf("DefaultConfig().Format = %v, want %v", config.Format, FormatText)
	}
	if config.Output != os.Stderr {
		t.Errorf("DefaultConfig().Output = %v, want os.Stderr", config.Output)
	}
}
