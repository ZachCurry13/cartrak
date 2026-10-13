package server

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestCleanupKeepsEverythingUnlessAskedAndNeverTouchesTrips(t *testing.T) {
	_, srv := healthServer(t, fakeProv{}, false, false)
	now := time.Now()
	srv.S.SavePoints(1, []store.Point{{TS: now.AddDate(0, 0, -200).Unix(), Lat: 1, Lon: 1}, {TS: now.Unix(), Lat: 2, Lon: 2}})
	end := now.AddDate(0, 0, -200).Unix() + 600
	tr := &store.Trip{StartTS: now.AddDate(0, 0, -200).Unix(), EndTS: &end, Category: "personal", CategorySource: "default"}
	srv.S.CreateTrip(tr)

	srv.cleanupOnce(now) // setting empty: keep everything
	if ps, _ := srv.S.PointsBetween(0, now.Unix()+1); len(ps) != 2 {
		t.Fatalf("nothing should be deleted by default: %d left", len(ps))
	}
	srv.Cfg.Set(settings.KeepDaysKey, "90")
	srv.cleanupOnce(now)
	if ps, _ := srv.S.PointsBetween(0, now.Unix()+1); len(ps) != 1 || ps[0].Lat != 2 {
		t.Fatalf("old points should go, new ones stay: %+v", ps)
	}
	if got, err := srv.S.GetTrip(tr.ID); err != nil || got == nil {
		t.Fatalf("the trip itself must stay: %v", err)
	}
}

func TestKeepDaysIsValidated(t *testing.T) {
	for _, bad := range []string{"abc", "-5", "99999", "1.5"} {
		if settings.Valid(settings.KeepDaysKey, bad) == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
	for _, good := range []string{"", "0", "30", "3650"} {
		if err := settings.Valid(settings.KeepDaysKey, good); err != nil {
			t.Fatalf("%q should be fine: %v", good, err)
		}
	}
}
