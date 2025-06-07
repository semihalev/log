package zlog

import (
	"bytes"
	"io"
	"testing"
)

func TestCompatibility(t *testing.T) {
	// Test KV methods
	t.Run("KVMethods", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewStructured()
		logger.SetWriter(&buf)

		// Test all KV methods
		logger.DebugKV("debug msg", "key1", "value1", "key2", 42)
		logger.InfoKV("info msg", "name", "john", "age", 30)
		logger.WarnKV("warn msg", "error", "timeout")
		logger.ErrorKV("error msg", "code", 500, "message", "internal error")

		// Should have logged something
		if buf.Len() == 0 {
			t.Error("No output from KV methods")
		}
	})

	// Test global functions with any values
	t.Run("GlobalCompatibility", func(t *testing.T) {
		original := Default()
		defer SetDefault(original)

		var buf bytes.Buffer
		logger := NewStructured()
		logger.SetWriter(&buf)
		SetDefault(logger)

		// Old style calls should work
		Info("simple message")
		Info("with values", "key", "value", "number", 123)
		Error("error occurred", "code", 404, "path", "/api/users")

		if buf.Len() == 0 {
			t.Error("No output from global functions")
		}
	})

	// Test SimpleLogger
	t.Run("SimpleLogger", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewSimple()
		logger.SetWriter(&buf)

		// Test variadic any
		logger.Info("hello", "world", 123)
		logger.Debug("debug", "message")
		logger.Warn("warning")
		logger.Error("error", "occurred")

		// Test formatted
		logger.Infof("Hello %s, you are %d years old", "John", 30)
		logger.Debugf("Debug: %v", map[string]int{"a": 1})
		logger.Warnf("Warning: %.2f%%", 95.5)
		logger.Errorf("Error %d: %s", 404, "not found")

		if buf.Len() == 0 {
			t.Error("No output from SimpleLogger")
		}
	})

	// Test Any field helper
	t.Run("AnyField", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewStructured()
		logger.SetWriter(&buf)

		// Mix Field types with Any
		logger.Info("test",
			String("name", "john"),
			Int("age", 30),
			Any("data", map[string]int{"a": 1, "b": 2}),
			Any("flag", true))

		if buf.Len() == 0 {
			t.Error("No output with Any field")
		}
	})
}

func TestGlobalCompatibilityMigration(t *testing.T) {
	// This test demonstrates v0.x compatibility
	original := Default()
	defer SetDefault(original)

	// Just verify the API works - content checking would need terminal decoder
	logger := NewStructured()
	logger.SetWriter(io.Discard)
	SetDefault(logger)

	// These should all work like v0.x without errors
	Info("Starting application")
	Debug("Debug mode enabled")
	Info("User logged in", "username", "john_doe", "user_id", 12345)
	Warn("Low memory", "available", "512MB", "threshold", "1GB")
	Error("Connection failed", "host", "db.example.com", "port", 5432, "error", "timeout")

	// If we got here without panic, the API is compatible
}

func TestSimpleLoggerFormatted(t *testing.T) {
	logger := NewSimple()
	logger.SetWriter(io.Discard)

	// Test formatted output - just verify no panic
	logger.Infof("Server started on %s:%d", "localhost", 8080)
	logger.Errorf("Failed to connect to %s: %v", "database", "connection refused")
	logger.Debugf("Debug %v", map[string]any{"test": 123})
	logger.Warnf("Warning: %d%%", 95)
}

// Benchmark compatibility layer
func BenchmarkCompatibilityKV(b *testing.B) {
	logger := NewStructured()
	logger.SetWriter(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoKV("test message", "key1", "value1", "key2", 42, "key3", true)
	}
}

func BenchmarkSimpleLogger(b *testing.B) {
	logger := NewSimple()
	logger.SetWriter(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message", "with", "values", 123)
	}
}

func BenchmarkSimpleLoggerSingle(b *testing.B) {
	logger := NewSimple()
	logger.SetWriter(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message")
	}
}

func BenchmarkCompatibilityCommonTypes(b *testing.B) {
	logger := NewStructured()
	logger.SetWriter(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoKV("test", "str", "value", "int", 42, "bool", true, "float", 3.14)
	}
}
