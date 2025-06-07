//go:build windows
// +build windows

package zlog

import (
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// MMapWriter provides zero-copy, zero-syscall logging via memory-mapped files
type MMapWriter struct {
	file       *os.File
	data       []byte
	size       int64
	offset     atomic.Int64
	pageSize   int64
	mapHandle  syscall.Handle
	fileHandle syscall.Handle
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

	// Get file handle
	fileHandle := syscall.Handle(file.Fd())

	// Create file mapping
	mapHandle, err := syscall.CreateFileMapping(
		fileHandle,
		nil,
		syscall.PAGE_READWRITE,
		uint32(size>>32),
		uint32(size),
		nil,
	)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Map view of file
	addr, err := syscall.MapViewOfFile(
		mapHandle,
		syscall.FILE_MAP_WRITE,
		0,
		0,
		uintptr(size),
	)
	if err != nil {
		syscall.CloseHandle(mapHandle)
		file.Close()
		return nil, err
	}

	// Create byte slice from mapped memory
	var data []byte
	header := (*[1 << 30]byte)(unsafe.Pointer(addr))
	data = header[:size:size]

	pageSize := int64(os.Getpagesize())

	return &MMapWriter{
		file:       file,
		data:       data,
		size:       size,
		pageSize:   pageSize,
		mapHandle:  mapHandle,
		fileHandle: fileHandle,
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
	// FlushViewOfFile for Windows
	syscall.FlushViewOfFile(uintptr(unsafe.Pointer(&w.data[offset])), uintptr(length))
}

// Close unmaps and closes the file
func (w *MMapWriter) Close() error {
	// Unmap view
	if err := syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&w.data[0]))); err != nil {
		return err
	}
	// Close mapping handle
	if err := syscall.CloseHandle(w.mapHandle); err != nil {
		return err
	}
	return w.file.Close()
}
