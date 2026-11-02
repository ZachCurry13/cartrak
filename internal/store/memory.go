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

// defaultConns is how many connections the database may use at once: a page asks for a dozen things together, and with one connection
// each waited for the one before it. SQLite in WAL mode lets readers run side by side (and a writer alongside them).
const defaultConns = 4

// CARTRAK_DB_CONNS changes it (1 to 8). 1 is the old behavior: everything in a single file.
func connCount() int {
	if n, err := strconv.Atoi(os.Getenv("CARTRAK_DB_CONNS")); err == nil && n >= 1 && n <= 8 {
		return n
	}
	return defaultConns
}

// memoryPragmas is the end of the connection string that makes SQLite use memory: a big page cache (a negative number is
// kibibytes), the file mapped into memory for reads, and temporary tables kept in memory too.
func memoryPragmas() string {
	mb := cacheMB()
	perConn := mb / connCount() // each connection has its own cache, so the total stays the size chosen
	if perConn < 4 {
		perConn = 4
	}
	return fmt.Sprintf("&_pragma=cache_size(-%d)&_pragma=mmap_size(%d)&_pragma=temp_store(MEMORY)", perConn*1024, mb*2*1024*1024) + "&_txlock=immediate" // a write transaction takes the write lock when it starts, so two never collide halfway
}
