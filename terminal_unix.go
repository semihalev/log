//go:build !windows

package zlog

import (
	"runtime"
	"syscall"
	"unsafe"
)

// isTerminal returns true if the file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios(), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

// ioctlReadTermios returns the ioctl to read terminal settings
func ioctlReadTermios() uintptr {
	switch runtime.GOOS {
	case "linux", "android":
		return 0x5401 // TCGETS
	case "darwin", "ios", "freebsd", "netbsd", "openbsd":
		return 0x40087468 // TIOCGETA
	case "solaris", "illumos":
		return 0x5401 // TCGETS
	default:
		return 0x5401 // Default to TCGETS
	}
}
