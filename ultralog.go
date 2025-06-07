// Package zlog provides the world's fastest logging library for Go
// Zero allocations, zero compromises, pure performance
package zlog

import (
	"encoding/binary"
	"io"
	"os"
	"sync/atomic"
	"time"
	"unsafe"
)

// Magic constants for binary format
const (
	MagicHeader = 0x554C4F47 // "ULOG"
	Version     = 1

	// Cache line size for padding
	CacheLineSize = 64
)

// Level represents log severity
type Level uint8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Logger is the core logger - exactly one cache line (64 bytes)
type Logger struct {
	level    Level          // 1 byte - current log level
	_        [7]byte        // 7 bytes padding for alignment
	writer   unsafe.Pointer // 8 bytes - *io.Writer
	sequence atomic.Uint64  // 8 bytes - Global sequence number
	_        [40]byte       // 40 bytes padding to 64 bytes
}

// Record represents a log entry - exactly one cache line (64 bytes)
type Record struct {
	Sequence uint64   // 8 bytes - unique sequence
	Time     uint64   // 8 bytes - unix nano timestamp
	Level    Level    // 1 byte
	_        [7]byte  // 7 bytes padding
	MsgLen   uint16   // 2 bytes - message length
	DataLen  uint16   // 2 bytes - data length
	_        [4]byte  // 4 bytes padding
	Data     [32]byte // 32 bytes - inline storage for small messages
}

// New creates a new ultra-fast logger
func New() *Logger {
	l := &Logger{}
	l.SetLevel(LevelInfo)
	l.SetWriter(os.Stdout)
	return l
}

// SetLevel sets the log level
// Note: This is not thread-safe - set level during initialization only
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// GetLevel gets the current log level
func (l *Logger) GetLevel() Level {
	return l.level
}

// SetWriter atomically sets the writer
func (l *Logger) SetWriter(w io.Writer) {
	atomic.StorePointer(&l.writer, unsafe.Pointer(&w))
}

// shouldLog inlined check for performance
//
//go:inline
func (l *Logger) shouldLog(level Level) bool {
	return l.level <= level
}

// getWriter gets the current writer
//
//go:inline
func (l *Logger) getWriter() io.Writer {
	return *(*io.Writer)(atomic.LoadPointer(&l.writer))
}

// logDirect logs directly without allocations
//
//go:noinline
func (l *Logger) logDirect(level Level, msg string) {
	// Stack allocated buffer
	var buf [256]byte
	pos := 0

	// Binary header
	binary.LittleEndian.PutUint32(buf[pos:], MagicHeader)
	pos += 4

	// Version
	buf[pos] = Version
	pos++

	// Level
	buf[pos] = byte(level)
	pos++

	// Sequence
	seq := l.sequence.Add(1)
	binary.LittleEndian.PutUint64(buf[pos:], seq)
	pos += 8

	// Timestamp
	now := uint64(time.Now().UnixNano())
	binary.LittleEndian.PutUint64(buf[pos:], now)
	pos += 8

	// Message length and data
	msgLen := len(msg)
	maxMsg := len(buf) - pos - 1 // Reserve 1 byte for length
	if maxMsg < 0 {
		maxMsg = 0
	}
	if msgLen > maxMsg {
		msgLen = maxMsg
	}
	if msgLen > 255 {
		msgLen = 255
	}
	buf[pos] = byte(msgLen)
	pos++

	// Copy message
	if msgLen > 0 && pos+msgLen <= len(buf) {
		copy(buf[pos:], msg[:msgLen])
		pos += msgLen
	}

	// Write
	l.getWriter().Write(buf[:pos])
}

// Debug logs a debug message
//
//go:inline
func (l *Logger) Debug(msg string) {
	if l.level <= LevelDebug {
		l.logDirect(LevelDebug, msg)
	}
}

// Info logs an info message
//
//go:inline
func (l *Logger) Info(msg string) {
	if l.level <= LevelInfo {
		l.logDirect(LevelInfo, msg)
	}
}

// Warn logs a warning message
//
//go:inline
func (l *Logger) Warn(msg string) {
	if l.level <= LevelWarn {
		l.logDirect(LevelWarn, msg)
	}
}

// Error logs an error message
//
//go:inline
func (l *Logger) Error(msg string) {
	if l.level <= LevelError {
		l.logDirect(LevelError, msg)
	}
}

// Fatal logs a fatal message and exits
//
//go:inline
func (l *Logger) Fatal(msg string) {
	if l.level <= LevelFatal {
		l.logDirect(LevelFatal, msg)
		os.Exit(1)
	}
}

// Built-in writers (kept for backward compatibility)

// StdoutWriter writes to stdout
var StdoutWriter io.Writer = os.Stdout

// StderrWriter writes to stderr
var StderrWriter io.Writer = os.Stderr

// DiscardWriter discards all output
var DiscardWriter io.Writer = io.Discard
