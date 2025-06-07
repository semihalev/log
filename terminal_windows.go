//go:build windows

package zlog

import (
	"os"
	"syscall"
	"unsafe"
)

// isTerminal returns true if the file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	var mode uint32
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")

	r, _, _ := getConsoleMode.Call(fd, uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

// Alternative simple check for standard outputs
func isTerminalSimple(fd uintptr) bool {
	return fd == uintptr(os.Stdout.Fd()) || fd == uintptr(os.Stderr.Fd())
}
