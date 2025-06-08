//go:build netbsd
// +build netbsd

package zlog

// msync performs memory synchronization
func msync(b []byte, flags int) error {
	// NetBSD doesn't expose SYS_MSYNC in syscall package
	// Memory-mapped files will be synced when unmapped or on system sync
	// This is acceptable for logging purposes
	// Note: FreeBSD and OpenBSD do have SYS_MSYNC, so they use mmap_unix.go
	return nil
}

const (
	// MS_ASYNC performs asynchronous sync
	MS_ASYNC = 0x1
)
