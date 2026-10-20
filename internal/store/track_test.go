package store

import (
	"testing"
	"time"
)

func openTemp(t *testing.T) *Store {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDeviceTokenIsOnlyKeptAsAHash(t *testing.T) {
	s := openTemp(t)
	id, err := s.CreateDevice("Phone", TokenHash("secret-token"))
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := s.DeviceByHash(TokenHash("secret-token")); d == nil || d.ID != id || d.Name != "Phone" {
		t.Fatalf("lookup by token hash failed: %+v", d)
	}
	if d, _ := s.DeviceByHash(TokenHash("wrong")); d != nil {
		t.Fatalf("a wrong token must not match: %+v", d)
	}
	var stored string
	s.DB.QueryRow(`SELECT token_hash FROM devices WHERE id=?`, id).Scan(&stored)
	if stored == "secret-token" || len(stored) != 64 {
		t.Fatalf("the token must not be stored as given: %q", stored)
	}
	if _, err := s.CreateDevice("Again", TokenHash("secret-token")); err == nil {
		t.Fatal("two devices with the same token must not exist")
	}
}

func TestPointsAreKeptWhenADeviceIsRemoved(t *testing.T) {
	s := openTemp(t)
	id, _ := s.CreateDevice("Phone", TokenHash("a"))
	spd := 50.0
	now := time.Now().Unix()
	if err := s.SavePoints(id, []Point{{TS: now, Lat: 1, Lon: 2, SpeedKph: &spd}, {TS: now + 5, Lat: 1.1, Lon: 2.1}}); err != nil {
		t.Fatal(err)
	}
	ds, _ := s.ListDevices()
	if len(ds) != 1 || ds[0].Points != 2 {
		t.Fatalf("devices = %+v", ds)
	}
	s.DeleteDevice(id)
	ps, _ := s.PointsBetween(now-1, now+10)
	if len(ps) != 2 || ps[0].SpeedKph == nil || *ps[0].SpeedKph != 50 || ps[1].SpeedKph != nil {
		t.Fatalf("points after removing the device: %+v", ps)
	}
	if only, _ := s.PointsBetween(now+1, now+10); len(only) != 1 {
		t.Fatalf("time window filter: %+v", only)
	}
}

func TestCleanupDeletesOnlyOldPoints(t *testing.T) {
	s := openTemp(t)
	id, _ := s.CreateDevice("Phone", TokenHash("a"))
	now := time.Now()
	s.SavePoints(id, []Point{{TS: now.Add(-100 * 24 * time.Hour).Unix(), Lat: 1, Lon: 1}, {TS: now.Unix(), Lat: 2, Lon: 2}})
	n, err := s.DeletePointsBefore(now.Add(-90 * 24 * time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("deleted %d, %v", n, err)
	}
	if ps, _ := s.PointsBetween(0, now.Unix()+1); len(ps) != 1 || ps[0].Lat != 2 {
		t.Fatalf("remaining: %+v", ps)
	}
}

func TestEventsNewestFirst(t *testing.T) {
	s := openTemp(t)
	s.AddEvent(Event{Device: 1, TS: 100, Type: "drive_start"})
	s.AddEvent(Event{Device: 1, TS: 200, Type: "drive_end"})
	es, _ := s.RecentEvents(10)
	if len(es) != 2 || es[0].Type != "drive_end" {
		t.Fatalf("events: %+v", es)
	}
}

func TestSampleCountsCountDifferentPositionsNotRepeats(t *testing.T) {
	s, _ := Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	for i, p := range [][2]float64{{44.1, -85.1}, {44.1, -85.1}, {44.1, -85.1}, {44.2, -85.2}} {
		if _, err := s.DB.Exec(`INSERT INTO samples(ts,lat,lon) VALUES(?,?,?)`, 1000+i*30, p[0], p[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DB.Exec(`INSERT INTO samples(ts) VALUES(2000)`); err != nil {
		t.Fatal(err)
	}
	n, d, err := s.SampleCounts(900, 3000)
	if err != nil || n != 5 || d != 2 {
		t.Fatalf("5 readings, 2 different positions: %d %d %v", n, d, err)
	}
}
