//go:build windows
// +build windows

package zlog

// msync is a no-op on Windows as memory-mapped files are handled differently
func msync(b []byte, flags int) error {
	// Windows automatically syncs memory-mapped files
	return nil
}

const (
	MS_ASYNC = 0x1
)
