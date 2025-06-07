package zlog

import (
	"sync/atomic"
	"unsafe"
)

// ZeroAllocLogger is the world's fastest logger - truly zero allocations
type ZeroAllocLogger struct {
	state    uint64         // atomic: level(8) | flags(8) | reserved(48)
	writer   unsafe.Pointer // *ZeroWriter
	sequence uint64         // atomic counter
	_        [40]byte       // padding to 64 bytes cache line
}

// ZeroWriter is a zero-allocation writer interface
type ZeroWriter interface {
	// WriteZero writes with zero allocations - buffer is reused
	WriteZero(buf *[256]byte, n int)
}

// DiscardZeroWriter discards all output with zero allocations
type DiscardZeroWriter struct{}

func (DiscardZeroWriter) WriteZero(*[256]byte, int) {}

// NewZeroAllocLogger creates the fastest possible logger
func NewZeroAllocLogger() *ZeroAllocLogger {
	l := &ZeroAllocLogger{}
	atomic.StoreUint64(&l.state, uint64(LevelInfo))
	w := ZeroWriter(DiscardZeroWriter{})
	atomic.StorePointer(&l.writer, unsafe.Pointer(&w))
	return l
}

// SetLevel atomically sets the log level
func (l *ZeroAllocLogger) SetLevel(level Level) {
	state := atomic.LoadUint64(&l.state)
	state = (state &^ 0xFF) | uint64(level)
	atomic.StoreUint64(&l.state, state)
}

// SetZeroWriter atomically sets the writer
func (l *ZeroAllocLogger) SetZeroWriter(w ZeroWriter) {
	atomic.StorePointer(&l.writer, unsafe.Pointer(&w))
}

// getWriter gets the current writer
//
//go:inline
func (l *ZeroAllocLogger) getWriter() ZeroWriter {
	return *(*ZeroWriter)(atomic.LoadPointer(&l.writer))
}

// shouldLog inlined check for performance
//
//go:inline
func (l *ZeroAllocLogger) shouldLog(level Level) bool {
	return Level(atomic.LoadUint64(&l.state)&0xFF) <= level
}

// Info logs an info message with zero allocations
//
//go:noinline
func (l *ZeroAllocLogger) Info(msg string) {
	if !l.shouldLog(LevelInfo) {
		return
	}

	// Stack allocated buffer - no heap allocation
	var buf [256]byte

	// Header
	*(*uint32)(unsafe.Pointer(&buf[0])) = MagicHeader
	buf[4] = Version
	buf[5] = byte(LevelInfo)

	// Sequence
	seq := atomic.AddUint64(&l.sequence, 1)
	*(*uint64)(unsafe.Pointer(&buf[6])) = seq

	// Timestamp - avoid time.Now() allocation
	*(*uint64)(unsafe.Pointer(&buf[14])) = uint64(nanotime())

	// Message
	msgLen := len(msg)
	if msgLen > 233 { // 256 - 23 header
		msgLen = 233
	}
	buf[22] = byte(msgLen)

	// Copy without creating slice
	for i := 0; i < msgLen; i++ {
		buf[23+i] = msg[i]
	}

	// Write with zero allocations
	l.getWriter().WriteZero(&buf, 23+msgLen)
}

// Debug logs a debug message
//
//go:noinline
func (l *ZeroAllocLogger) Debug(msg string) {
	if !l.shouldLog(LevelDebug) {
		return
	}
	l.logLevel(LevelDebug, msg)
}

// Warn logs a warning message
//
//go:noinline
func (l *ZeroAllocLogger) Warn(msg string) {
	if !l.shouldLog(LevelWarn) {
		return
	}
	l.logLevel(LevelWarn, msg)
}

// Error logs an error message
//
//go:noinline
func (l *ZeroAllocLogger) Error(msg string) {
	if !l.shouldLog(LevelError) {
		return
	}
	l.logLevel(LevelError, msg)
}

// logLevel is the common logging function
//
//go:noinline
func (l *ZeroAllocLogger) logLevel(level Level, msg string) {
	var buf [256]byte

	*(*uint32)(unsafe.Pointer(&buf[0])) = MagicHeader
	buf[4] = Version
	buf[5] = byte(level)

	seq := atomic.AddUint64(&l.sequence, 1)
	*(*uint64)(unsafe.Pointer(&buf[6])) = seq
	*(*uint64)(unsafe.Pointer(&buf[14])) = uint64(nanotime())

	msgLen := len(msg)
	if msgLen > 233 {
		msgLen = 233
	}
	buf[22] = byte(msgLen)

	for i := 0; i < msgLen; i++ {
		buf[23+i] = msg[i]
	}

	l.getWriter().WriteZero(&buf, 23+msgLen)
}

// nanotime returns current time in nanoseconds without allocation
//
//go:linkname nanotime runtime.nanotime
func nanotime() int64
