package store

import (
	"fmt"
	"os"
	"strconv"
)

// defaultCacheMB is how much of the database SQLite keeps in memory. Its own default is about 2 MB, so on a database of hundreds of
// megabytes nearly every read goes back to the disk. A home server has memory to spare, and the cache only fills as pages are read.
const defaultCacheMB = 256

// CARTRAK_DB_CACHE_MB changes it (8 to 4096).
func cacheMB() int {
	if n, err := strconv.Atoi(os.Getenv("CARTRAK_DB_CACHE_MB")); err == nil && n >= 8 && n <= 4096 {
		return n
	}
	return defaultCacheMB
}

// memoryPragmas is the end of the connection string that makes SQLite use memory: a big page cache (a negative number is
// kibibytes), the file mapped into memory for reads, and temporary tables kept in memory too.
func memoryPragmas() string {
	mb := cacheMB()
	return fmt.Sprintf("&_pragma=cache_size(-%d)&_pragma=mmap_size(%d)&_pragma=temp_store(MEMORY)", mb*1024, mb*2*1024*1024)
}
