//go:build linux

// Package diskfree says how much room is left on the disk that holds CarTrak's data. It works on Linux (the
// Docker image); elsewhere it reports that it does not know.
package diskfree

import "syscall"

// Space returns the free and total bytes of the disk holding path, and false if that cannot be found out.
func Space(path string) (free, total uint64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	return st.Bavail * uint64(st.Bsize), st.Blocks * uint64(st.Bsize), true
}
