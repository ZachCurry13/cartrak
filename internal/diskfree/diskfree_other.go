//go:build !linux

package diskfree

// Space is not available on this system.
func Space(path string) (free, total uint64, ok bool) { return 0, 0, false }
