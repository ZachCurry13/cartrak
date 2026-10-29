package store

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/remotestart"
)

// Going over the same readings twice (the background pass after a restart) must not record a remote start twice, and a database
// that already holds such a pair is tidied the next time it is opened.
func TestARemoteStartIsNeverRecordedTwiceAndOldPairsAreTidied(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoteStartsBetween(0, 1<<40); err != nil { // makes the table
		t.Fatal(err)
	}
	a := &remotestart.Session{Start: 1000, End: 2300}
	s.saveRemote(a)
	b := &remotestart.Session{Start: 1010, End: 2300} // the same start seen again, a few seconds off
	s.saveRemote(b)
	if got, _ := s.RemoteStartsBetween(0, 1<<40); len(got) != 1 {
		t.Fatalf("one start, not two: %+v", got)
	}
	// a pair left by an earlier version
	if _, err := s.DB.Exec(`INSERT INTO remote_starts(start_ts,end_ts,restarts,drove) VALUES(5000,6000,0,0),(5000,6000,0,0)`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	got, _ := s.RemoteStartsBetween(0, 1<<40)
	if len(got) != 2 || got[0].Start != 1000 || got[1].Start != 5000 {
		t.Fatalf("the old pair is one now: %+v", got)
	}
}
