package zlog

import (
	"testing"
)

func BenchmarkZeroAllocLogger(b *testing.B) {
	logger := NewZeroAllocLogger()
	logger.SetZeroWriter(DiscardZeroWriter{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkZeroAllocLoggerParallel(b *testing.B) {
	logger := NewZeroAllocLogger()
	logger.SetZeroWriter(DiscardZeroWriter{})

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message")
		}
	})
}

// Benchmark the absolute minimum - just the level check
func BenchmarkZeroAllocLoggerDisabled(b *testing.B) {
	logger := NewZeroAllocLogger()
	logger.SetLevel(LevelError) // Info is disabled

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("this won't be logged")
	}
}
