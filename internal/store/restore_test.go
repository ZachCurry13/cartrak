package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeStore(t *testing.T, dir, user string, trips int) *Store {
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateUser(user, "hash", "admin"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < trips; i++ {
		if err := s.CreateTrip(&Trip{StartTS: int64(1000 + i)}); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestABackupIsInspectedThenSwappedInAtStartWithTheOldDataKept(t *testing.T) {
	oldDir, newDir := t.TempDir(), t.TempDir()
	cur := makeStore(t, oldDir, "now", 1)
	src := makeStore(t, newDir, "then", 3)
	backup := filepath.Join(t.TempDir(), "b.db")
	if err := src.Backup(backup, true); err != nil {
		t.Fatal(err)
	}
	src.Close()

	info, err := InspectBackup(backup)
	if err != nil || info.Trips != 3 || len(info.Users) != 1 || info.Users[0] != "then" {
		t.Fatalf("%+v %v", info, err)
	}
	junk := filepath.Join(t.TempDir(), "x.db")
	os.WriteFile(junk, []byte("this is not a database at all, just text"), 0o644)
	if _, err := InspectBackup(junk); err == nil || !strings.Contains(err.Error(), "not a CarTrak backup") {
		t.Fatalf("junk is refused: %v", err)
	}

	// The owner confirmed: the file waits as pending.db with the flag, and the current database is closed.
	dir := RestoreDir(oldDir)
	os.MkdirAll(dir, 0o755)
	data, _ := os.ReadFile(backup)
	os.WriteFile(filepath.Join(dir, PendingName), data, 0o644)
	if done, _ := ApplyPendingRestore(oldDir); done {
		t.Fatal("without the confirmation flag nothing happens")
	}
	os.WriteFile(filepath.Join(dir, ApplyFlag), []byte("1"), 0o644)
	cur.Checkpoint()
	cur.Close()
	done, err := ApplyPendingRestore(oldDir)
	if err != nil || !done {
		t.Fatalf("applied: %v %v", done, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ApplyFlag)); err == nil {
		t.Fatal("the flag is removed so it is never applied twice")
	}
	kept, _ := filepath.Glob(filepath.Join(oldDir, "cartrak.db.before-restore-*"))
	if len(kept) != 1 {
		t.Fatalf("the current data is kept: %v", kept)
	}
	restored, err := Open(oldDir)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if ts, _ := restored.ListTrips(TripFilter{}); len(ts) != 3 {
		t.Fatalf("the backup's data is in place: %d trips", len(ts))
	}
	if u, _ := restored.UserByName("then"); u == nil {
		t.Fatal("the backup's accounts are in place")
	}
}
