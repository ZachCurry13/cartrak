package store

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func countdown(left float64) func(*ford.State) {
	return func(st *ford.State) {
		st.Extra["telemetry.metrics.remoteStartCountdownTimer.value"] = left
		st.Extra["telemetry.metrics.remoteStartRunDurationSetting.value"] = 600.0
	}
}

func TestRemoteStartsAreRecordedAsTheyHappen(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 1, 10, 7, 0, 0, 0, time.UTC)
	for i, left := range []float64{0, 540, 240, 0, 0} {
		if _, err := s.SaveState("t", state(base.Add(time.Duration(i)*5*time.Minute), 80-float64(i), countdown(left))); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.RemoteStartsBetween(base.Unix(), base.Add(time.Hour).Unix())
	if err != nil || len(got) != 1 {
		t.Fatalf("one remote start: %v %+v", err, got)
	}
	r := got[0]
	if r.Start != base.Unix()+300-60 || r.End != base.Unix()+600+240 || *r.StartSoC != 80 || *r.EndSoC != 77 || *r.TempC != 15 {
		t.Fatalf("recorded: %+v", r)
	}
	if got, _ := s.RemoteStartsBetween(base.Add(2*time.Hour).Unix(), base.Add(3*time.Hour).Unix()); len(got) != 0 {
		t.Fatal("only the ones that overlap")
	}
}

func TestOldRemoteStartsAreFoundOnce(t *testing.T) {
	s := openTemp(t)
	base := time.Date(2026, 1, 10, 7, 0, 0, 0, time.UTC)
	// saved without the live check, as an older version would have
	for i, left := range []float64{0, 500, 0, 0, 450, 100, 0} {
		if _, err := s.DB.Exec(`INSERT INTO samples(ts,soc_pct,extra) VALUES(?,?,?)`, base.Add(time.Duration(i)*5*time.Minute).Unix(), 70, `{"m.remoteStartCountdownTimer.value":`+strconv.FormatFloat(left, 'f', -1, 64)+`}`); err != nil {
			t.Fatal(err)
		}
	}
	upTo := s.LatestSampleID()
	if err := s.BackfillRemoteStarts(context.Background(), upTo, 3, 0); err != nil {
		t.Fatal(err)
	}
	_ = s.BackfillRemoteStarts(context.Background(), upTo, 3, 0) // a second run does nothing
	if got, _ := s.RemoteStartsBetween(0, base.Add(time.Hour).Unix()); len(got) != 2 {
		t.Fatalf("two old remote starts, once each: %+v", got)
	}
}
