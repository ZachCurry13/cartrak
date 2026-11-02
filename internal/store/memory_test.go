package store

import "testing"

func TestTheDatabaseKeepsALargePartOfItselfInMemory(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var cache int
	if err := s.DB.QueryRow(`PRAGMA cache_size`).Scan(&cache); err != nil || cache != -64*1024 {
		t.Fatalf("a 256 MB page cache shared by 4 connections, so 64 MB each: %d %v", cache, err)
	}
	var temp int
	if err := s.DB.QueryRow(`PRAGMA temp_store`).Scan(&temp); err != nil || temp != 2 {
		t.Fatalf("temporary tables in memory: %d %v", temp, err)
	}
	t.Setenv("CARTRAK_DB_CACHE_MB", "64")
	s2, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if err := s2.DB.QueryRow(`PRAGMA cache_size`).Scan(&cache); err != nil || cache != -16*1024 {
		t.Fatalf("the size can be chosen (64 MB over 4 connections): %d %v", cache, err)
	}
	t.Setenv("CARTRAK_DB_CACHE_MB", "99999")
	if cacheMB() != defaultCacheMB {
		t.Fatal("a silly size falls back to the default")
	}
}
