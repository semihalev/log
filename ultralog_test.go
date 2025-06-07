package zlog

import (
	"bytes"
	"testing"
)

func TestUltralogBasic(t *testing.T) {
	// Test basic logging
	var buf bytes.Buffer
	logger := New()
	logger.SetWriter(&buf)

	logger.Info("test message")

	if buf.Len() == 0 {
		t.Fatal("Expected log output, got none")
	}
}

func TestTerminalWriter(t *testing.T) {
	// For now, just test that we can create terminal writers
	stdout := StdoutTerminal()
	if stdout == nil {
		t.Fatal("StdoutTerminal returned nil")
	}

	stderr := StderrTerminal()
	if stderr == nil {
		t.Fatal("StderrTerminal returned nil")
	}
}

func TestStructuredLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStructured()
	logger.SetWriter(&buf)

	logger.Info("test", String("key", "value"), Int("num", 42))

	if buf.Len() == 0 {
		t.Fatal("Expected log output, got none")
	}
}
