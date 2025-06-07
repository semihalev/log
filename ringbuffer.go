package zlog

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// RingBuffer is a lock-free single producer, multiple consumer ring buffer
type RingBuffer struct {
	_      [CacheLineSize]byte // Padding
	mask   uint64              // Size mask for fast modulo
	_      [56]byte            // Padding to cache line
	head   atomic.Uint64       // Producer position
	_      [56]byte            // Padding to cache line
	tail   atomic.Uint64       // Consumer position
	_      [56]byte            // Padding to cache line
	buffer []unsafe.Pointer    // Buffer of pointers
	size   int                 // Buffer size
}

// NewRingBuffer creates a new ring buffer with the given size (must be power of 2)
func NewRingBuffer(size int) *RingBuffer {
	// Ensure size is power of 2
	if size&(size-1) != 0 {
		panic("ring buffer size must be power of 2")
	}

	rb := &RingBuffer{
		buffer: make([]unsafe.Pointer, size),
		size:   size,
		mask:   uint64(size - 1),
	}

	return rb
}

// Entry represents a log entry in the ring buffer
type Entry struct {
	data [256]byte // Fixed size entry
	len  int
}

// Put adds data to the ring buffer (single producer)
func (rb *RingBuffer) Put(data []byte) bool {
	// Get next position
	head := rb.head.Load()
	next := (head + 1) & rb.mask

	// Check if full
	if next == rb.tail.Load() {
		return false
	}

	// Copy data to entry
	entry := &Entry{}
	entry.len = copy(entry.data[:], data)

	// Store entry
	atomic.StorePointer(&rb.buffer[head], unsafe.Pointer(entry))

	// Update head
	rb.head.Store(next)
	return true
}

// Get retrieves data from the ring buffer (multiple consumers)
func (rb *RingBuffer) Get() ([]byte, bool) {
	for {
		tail := rb.tail.Load()
		head := rb.head.Load()

		// Empty?
		if tail == head {
			return nil, false
		}

		// Try to claim this slot
		next := (tail + 1) & rb.mask
		if rb.tail.CompareAndSwap(tail, next) {
			// Wait for data to be available
			for {
				p := atomic.LoadPointer(&rb.buffer[tail])
				if p != nil {
					entry := (*Entry)(p)
					// Clear slot
					atomic.StorePointer(&rb.buffer[tail], nil)
					return entry.data[:entry.len], true
				}
				runtime.Gosched()
			}
		}
	}
}

// AsyncWriter wraps a writer with a ring buffer for async operation
type AsyncWriter struct {
	rb     *RingBuffer
	writer Writer
	done   chan struct{}
}

// NewAsyncWriter creates a new async writer
func NewAsyncWriter(w Writer, bufferSize int) *AsyncWriter {
	aw := &AsyncWriter{
		rb:     NewRingBuffer(bufferSize),
		writer: w,
		done:   make(chan struct{}),
	}

	// Start consumer
	go aw.consumer()

	return aw
}

// consumer processes entries from the ring buffer
func (aw *AsyncWriter) consumer() {
	for {
		select {
		case <-aw.done:
			// Drain remaining entries
			for {
				data, ok := aw.rb.Get()
				if !ok {
					return
				}
				aw.writer(data)
			}
		default:
			data, ok := aw.rb.Get()
			if ok {
				aw.writer(data)
			} else {
				runtime.Gosched()
			}
		}
	}
}

// Write adds data to the ring buffer
func (aw *AsyncWriter) Write(b []byte) error {
	// Try to put in ring buffer
	if aw.rb.Put(b) {
		return nil
	}

	// Buffer full - write directly (backpressure)
	return aw.writer(b)
}

// Close stops the async writer
func (aw *AsyncWriter) Close() error {
	close(aw.done)
	return nil
}

// Writer returns a Writer function for the logger
func (aw *AsyncWriter) Writer() Writer {
	return func(b []byte) error {
		return aw.Write(b)
	}
}
