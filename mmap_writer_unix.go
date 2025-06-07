//go:build !windows
// +build !windows

package zlog

import (
	"os"
	"sync/atomic"
	"syscall"
)

// MMapWriter provides zero-copy, zero-syscall logging via memory-mapped files
type MMapWriter struct {
	file     *os.File
	data     []byte
	size     int64
	offset   atomic.Int64
	pageSize int64
}

// NewMMapWriter creates a new memory-mapped file writer
func NewMMapWriter(path string, size int64) (*MMapWriter, error) {
	// Create or open file
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// Resize file
	if err := file.Truncate(size); err != nil {
		file.Close()
		return nil, err
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, int(size),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, err
	}

	pageSize := int64(os.Getpagesize())

	return &MMapWriter{
		file:     file,
		data:     data,
		size:     size,
		pageSize: pageSize,
	}, nil
}

// Write writes data to the memory-mapped file
func (w *MMapWriter) Write(b []byte) (int, error) {
	n := int64(len(b))
	if n == 0 {
		return 0, nil
	}

	// Get current offset and advance
	offset := w.offset.Add(n)
	if offset > w.size {
		// Wrap around (circular buffer)
		w.offset.Store(n)
		offset = n
	}
	start := offset - n

	// Direct memory copy - no syscalls!
	copy(w.data[start:offset], b)

	// Only sync if we cross a page boundary
	startPage := start / w.pageSize
	endPage := offset / w.pageSize
	if startPage != endPage {
		// Async sync in background
		go w.syncRange(startPage*w.pageSize, w.pageSize)
	}

	return len(b), nil
}

// syncRange asynchronously syncs a range of memory
func (w *MMapWriter) syncRange(offset, length int64) {
	if offset+length > w.size {
		length = w.size - offset
	}
	// MS_ASYNC = non-blocking sync
	msync(w.data[offset:offset+length], MS_ASYNC)
}

// Close unmaps and closes the file
func (w *MMapWriter) Close() error {
	if err := syscall.Munmap(w.data); err != nil {
		return err
	}
	return w.file.Close()
}
