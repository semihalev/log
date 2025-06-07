package zlog

import (
	"bytes"
	"strings"
	"testing"
)

func TestGlobalLogger(t *testing.T) {
	// Save original logger
	original := Default()
	defer SetDefault(original)

	// Create a buffer to capture output
	var buf bytes.Buffer
	captureWriter := func(b []byte) error {
		buf.Write(b)
		return nil
	}

	// Create new logger with custom writer
	logger := NewStructured()
	logger.SetWriter(captureWriter)
	SetDefault(logger)

	// Test global functions
	Debug("debug message", String("key", "value"))
	Info("info message", Int("count", 42))
	Warn("warn message", Bool("flag", true))
	Error("error message", Float64("pi", 3.14159))

	// Verify output contains expected content
	output := buf.String()
	if len(output) == 0 {
		t.Error("No output captured")
	}

	// Since we're using binary format, just verify we got data
	if buf.Len() < 100 {
		t.Error("Output too short")
	}
}

func TestGlobalSetLevel(t *testing.T) {
	// Save original logger
	original := Default()
	defer SetDefault(original)

	// Create a buffer to capture output
	var buf bytes.Buffer
	captureWriter := func(b []byte) error {
		buf.Write(b)
		return nil
	}

	// Create new logger
	logger := NewStructured()
	logger.SetWriter(captureWriter)
	SetDefault(logger)

	// Set level to Error
	SetLevel(LevelError)

	// These should not log
	buf.Reset()
	Debug("debug")
	Info("info")
	Warn("warn")

	if buf.Len() > 0 {
		t.Error("Lower level messages were logged")
	}

	// This should log
	buf.Reset()
	Error("error")

	if buf.Len() == 0 {
		t.Error("Error message was not logged")
	}
}

func TestGlobalSetWriter(t *testing.T) {
	// Save original logger
	original := Default()
	defer SetDefault(original)

	// Create new logger
	logger := NewStructured()
	SetDefault(logger)

	// Create a buffer to capture output
	var buf bytes.Buffer
	captureWriter := func(b []byte) error {
		buf.Write(b)
		return nil
	}

	// Set global writer
	SetWriter(captureWriter)

	// Log something
	Info("test message")

	// Verify output
	if buf.Len() == 0 {
		t.Error("No output captured after SetWriter")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Test that default logger is initialized
	logger := Default()
	if logger == nil {
		t.Fatal("Default logger is nil")
	}

	// Test that we can use it immediately
	logger.Info("test")
}

func TestGlobalFatal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fatal test in short mode")
	}

	// Run in subprocess to test exit
	if strings.Contains(strings.Join(callStack(), " "), "TestGlobalFatal") {
		Fatal("fatal error", String("test", "value"))
		return
	}

	// This test is similar to TestFatal in fatal_test.go
	// It would need subprocess testing to verify exit behavior
}

// Helper to get call stack
func callStack() []string {
	var stack []string
	// Simplified - in real implementation would use runtime.Callers
	return stack
}
