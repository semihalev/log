package zlog

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func BenchmarkTerminalWriter(b *testing.B) {
	tw := NewTerminalWriter(io.Discard)
	l := New()
	l.SetWriter(tw)
	l.SetLevel(LevelInfo)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("The quick brown fox jumps over the lazy dog")
	}
}

func BenchmarkTerminalWriterWithFields(b *testing.B) {
	// Create binary log data with fields
	var buf bytes.Buffer
	tmpLogger := NewStructured()
	tmpLogger.SetWriter(&buf)
	tmpLogger.SetLevel(LevelInfo)

	tmpLogger.Info("User logged in",
		String("username", "john_doe"),
		Int("user_id", 42),
		String("ip", "192.168.1.1"),
		Bool("success", true))

	data := buf.Bytes()

	// Benchmark terminal writer
	tw := NewTerminalWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.Write(data)
	}
}

func BenchmarkTerminalWriterColorOutput(b *testing.B) {
	// Create binary log data
	var buf bytes.Buffer
	tmpLogger := NewStructured()
	tmpLogger.SetWriter(&buf)
	tmpLogger.SetLevel(LevelInfo)

	tmpLogger.Info("Colored output test",
		String("status", "active"),
		Int("count", 100))

	data := buf.Bytes()

	// Force color mode
	tw := &TerminalWriter{
		out:        io.Discard,
		useColor:   true,
		timeFormat: termTimeFormat,
		buf:        make([]byte, 0, 2048),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.Write(data)
	}
}

func BenchmarkTerminalWriterEscapeString(b *testing.B) {
	// Create binary log data with string that needs escaping
	var buf bytes.Buffer
	tmpLogger := NewStructured()
	tmpLogger.SetWriter(&buf)
	tmpLogger.SetLevel(LevelInfo)

	tmpLogger.Info("Test message",
		String("escaped", "This \"string\" has\nspecial\tcharacters"))

	data := buf.Bytes()

	tw := NewTerminalWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.Write(data)
	}
}

func BenchmarkTerminalWriterLongMessage(b *testing.B) {
	// Create binary log data
	var buf bytes.Buffer
	tmpLogger := NewStructured()
	tmpLogger.SetWriter(&buf)
	tmpLogger.SetLevel(LevelInfo)

	longMsg := "This is a very long message that might cause buffer reallocation when formatting"
	tmpLogger.Info(longMsg,
		String("key1", "value1"),
		String("key2", "value2"),
		String("key3", "value3"))

	data := buf.Bytes()

	tw := NewTerminalWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.Write(data)
	}
}

func BenchmarkTerminalWriterManyFields(b *testing.B) {
	// Create binary log data with many fields
	var buf bytes.Buffer
	tmpLogger := NewStructured()
	tmpLogger.SetWriter(&buf)
	tmpLogger.SetLevel(LevelInfo)

	tmpLogger.Info("Many fields",
		String("field1", "value1"),
		String("field2", "value2"),
		String("field3", "value3"),
		String("field4", "value4"),
		String("field5", "value5"),
		Int("num1", 100),
		Int("num2", 200),
		Int("num3", 300),
		Float64("float1", 3.14159),
		Float64("float2", 2.71828))

	data := buf.Bytes()

	tw := NewTerminalWriter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tw.Write(data)
	}
}

// Test escape string performance specifically
func BenchmarkEscapeString(b *testing.B) {
	cases := []struct {
		name string
		str  string
	}{
		{"NoEscape", "simple string"},
		{"WithSpaces", "string with spaces"},
		{"WithQuotes", `string "with" quotes`},
		{"WithNewlines", "string\nwith\nnewlines"},
		{"WithTabs", "string\twith\ttabs"},
		{"Mixed", `complex "string" with\nnewlines\tand\rcarriage returns`},
		{"Long", "very long string that might exceed stack buffer very long string that might exceed stack buffer very long string"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = escapeString(tc.str)
			}
		})
	}
}

// Benchmark different sizes
func BenchmarkTerminalWriterSize(b *testing.B) {
	sizes := []int{1, 5, 10, 20}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Fields%d", size), func(b *testing.B) {
			// Create binary log data
			var buf bytes.Buffer
			tmpLogger := NewStructured()
			tmpLogger.SetWriter(&buf)
			tmpLogger.SetLevel(LevelInfo)

			fields := make([]Field, size)
			for i := 0; i < size; i++ {
				fields[i] = String(fmt.Sprintf("field%d", i), fmt.Sprintf("value%d", i))
			}

			tmpLogger.Info("Test message", fields...)
			data := buf.Bytes()

			tw := NewTerminalWriter(io.Discard)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tw.Write(data)
			}
		})
	}
}

// Benchmark vs zerolog terminal writer equivalent
func BenchmarkTerminalWriterComparison(b *testing.B) {
	b.Run("zlog", func(b *testing.B) {
		tw := NewTerminalWriter(io.Discard)
		l := NewStructured()
		l.SetWriter(tw)
		l.SetLevel(LevelInfo)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Info("Request handled",
				String("method", "POST"),
				String("path", "/api/users"),
				Int("status", 200),
				Float64("duration", 1.234))
		}
	})

	b.Run("zlog-direct-write", func(b *testing.B) {
		// Create binary log data once
		var buf bytes.Buffer
		tmpLogger := NewStructured()
		tmpLogger.SetWriter(&buf)
		tmpLogger.SetLevel(LevelInfo)

		tmpLogger.Info("Request handled",
			String("method", "POST"),
			String("path", "/api/users"),
			Int("status", 200),
			Float64("duration", 1.234))

		data := buf.Bytes()
		tw := NewTerminalWriter(io.Discard)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tw.Write(data)
		}
	})
}
