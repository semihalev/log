//go:build !windows
// +build !windows

package zlog

import (
	"syscall"
	"unsafe"
)

// msync performs memory synchronization
func msync(b []byte, flags int) error {
	// On most Unix systems including macOS, Msync is available
	// but the function signature might vary
	_, _, err := syscall.Syscall(syscall.SYS_MSYNC,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		uintptr(flags))
	if err != 0 {
		return err
	}
	return nil
}

const (
	// MS_ASYNC performs asynchronous sync
	MS_ASYNC = 0x1
)
