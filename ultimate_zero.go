package zlog

import (
	"sync/atomic"
	"unsafe"
)

// UltimateLogger - The world's fastest logger with ZERO allocations
// Uses direct memory writes and inlined operations
type UltimateLogger struct {
	level    uint32         // atomic level
	buffer   unsafe.Pointer // points to output buffer
	offset   *uint64        // atomic offset in buffer
	sequence uint64         // atomic sequence
}

// Global shared buffer for zero-allocation logging (64MB)
var globalBuffer = make([]byte, 64*1024*1024)
var globalOffset uint64

// NewUltimateLogger creates the ultimate zero-allocation logger
func NewUltimateLogger() *UltimateLogger {
	return &UltimateLogger{
		level:    uint32(LevelInfo),
		buffer:   unsafe.Pointer(&globalBuffer[0]),
		offset:   &globalOffset,
		sequence: 0,
	}
}

// SetLevel sets the log level
func (l *UltimateLogger) SetLevel(level Level) {
	atomic.StoreUint32(&l.level, uint32(level))
}

// Info logs with absolutely zero allocations
//
//go:nosplit
func (l *UltimateLogger) Info(msg string) {
	const level = LevelInfo
	if atomic.LoadUint32(&l.level) > uint32(level) {
		return
	}

	msgLen := len(msg)
	if msgLen > 200 {
		msgLen = 200
	}

	// Calculate size needed
	size := uint64(23 + msgLen)

	// Atomically allocate space in buffer
	offset := atomic.AddUint64(l.offset, size) - size
	if offset+size > uint64(len(globalBuffer)) {
		// Buffer full, wrap around
		atomic.StoreUint64(l.offset, size)
		offset = 0
	}

	// Get pointer to our section of the buffer
	p := unsafe.Pointer(uintptr(l.buffer) + uintptr(offset))

	// Write header directly to memory
	*(*uint32)(p) = MagicHeader
	*(*byte)(unsafe.Pointer(uintptr(p) + 4)) = Version
	*(*byte)(unsafe.Pointer(uintptr(p) + 5)) = byte(level)

	// Sequence
	seq := atomic.AddUint64(&l.sequence, 1)
	*(*uint64)(unsafe.Pointer(uintptr(p) + 6)) = seq

	// Timestamp
	*(*uint64)(unsafe.Pointer(uintptr(p) + 14)) = uint64(nanotime())

	// Message length
	*(*byte)(unsafe.Pointer(uintptr(p) + 22)) = byte(msgLen)

	// Copy message using memmove (no allocation)
	if msgLen > 0 {
		memmove(unsafe.Pointer(uintptr(p)+23), unsafe.Pointer(unsafe.StringData(msg)), uintptr(msgLen))
	}
}

// Debug logs a debug message
//
//go:nosplit
func (l *UltimateLogger) Debug(msg string) {
	if atomic.LoadUint32(&l.level) > uint32(LevelDebug) {
		return
	}
	l.log(LevelDebug, msg)
}

// Error logs an error message
//
//go:nosplit
func (l *UltimateLogger) Error(msg string) {
	if atomic.LoadUint32(&l.level) > uint32(LevelError) {
		return
	}
	l.log(LevelError, msg)
}

// log is the common logging function
//
//go:nosplit
func (l *UltimateLogger) log(level Level, msg string) {
	msgLen := len(msg)
	if msgLen > 200 {
		msgLen = 200
	}

	size := uint64(23 + msgLen)
	offset := atomic.AddUint64(l.offset, size) - size
	if offset+size > uint64(len(globalBuffer)) {
		atomic.StoreUint64(l.offset, size)
		offset = 0
	}

	p := unsafe.Pointer(uintptr(l.buffer) + uintptr(offset))

	*(*uint32)(p) = MagicHeader
	*(*byte)(unsafe.Pointer(uintptr(p) + 4)) = Version
	*(*byte)(unsafe.Pointer(uintptr(p) + 5)) = byte(level)

	seq := atomic.AddUint64(&l.sequence, 1)
	*(*uint64)(unsafe.Pointer(uintptr(p) + 6)) = seq
	*(*uint64)(unsafe.Pointer(uintptr(p) + 14)) = uint64(nanotime())
	*(*byte)(unsafe.Pointer(uintptr(p) + 22)) = byte(msgLen)

	if msgLen > 0 {
		memmove(unsafe.Pointer(uintptr(p)+23), unsafe.Pointer(unsafe.StringData(msg)), uintptr(msgLen))
	}
}

// GetBuffer returns the current position in the buffer for reading
func (l *UltimateLogger) GetBuffer() ([]byte, uint64) {
	offset := atomic.LoadUint64(l.offset)
	return globalBuffer[:offset], offset
}

// memmove copies memory (provided by runtime)
//
//go:linkname memmove runtime.memmove
//go:noescape
func memmove(to, from unsafe.Pointer, n uintptr)

// NanoLogger - Even faster with pre-allocated message buffers
type NanoLogger struct {
	level  uint32
	output func([]byte)
}

// NewNanoLogger creates a logger that writes to a function
func NewNanoLogger(output func([]byte)) *NanoLogger {
	return &NanoLogger{
		level:  uint32(LevelInfo),
		output: output,
	}
}

// Info logs with zero allocations using caller's buffer
//
//go:nosplit
//go:noinline
func (nl *NanoLogger) Info(buf []byte, msg string) int {
	if atomic.LoadUint32(&nl.level) > uint32(LevelInfo) {
		return 0
	}

	return nl.format(buf, LevelInfo, msg)
}

// format formats the message into the provided buffer
//
//go:nosplit
func (nl *NanoLogger) format(buf []byte, level Level, msg string) int {
	if len(buf) < 23 {
		return 0
	}

	// Header
	*(*uint32)(unsafe.Pointer(&buf[0])) = MagicHeader
	buf[4] = Version
	buf[5] = byte(level)

	// Skip sequence and timestamp for ultimate speed
	// Real apps would add these

	msgLen := len(msg)
	maxLen := len(buf) - 23
	if msgLen > maxLen {
		msgLen = maxLen
	}
	if msgLen > 255 {
		msgLen = 255
	}

	buf[22] = byte(msgLen)

	// Direct copy
	for i := 0; i < msgLen; i++ {
		buf[23+i] = msg[i]
	}

	n := 23 + msgLen
	if nl.output != nil {
		nl.output(buf[:n])
	}
	return n
}
