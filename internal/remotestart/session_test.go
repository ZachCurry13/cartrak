package remotestart

import "testing"

func fp(v float64) *float64 { return &v }

func run(rs []Reading) (open *Session, done []Session) {
	var before *Reading
	for i := range rs {
		var ended *Session
		open, ended = Step(open, rs[i], before)
		if ended != nil {
			done = append(done, *ended)
		}
		before = &rs[i]
	}
	return open, done
}

func TestARemoteStartBecomesOneEvent(t *testing.T) {
	rs := []Reading{
		{TS: 1000, SoC: fp(80), Kwh: fp(70), TempC: fp(-5)},
		{TS: 1300, Left: 480, RunFor: 600, SoC: fp(80), Kwh: fp(69.8), TempC: fp(-5)}, // began 2 minutes before it was seen
		{TS: 1600, Left: 180, RunFor: 600, SoC: fp(79), Kwh: fp(69.1)},
		{TS: 1900, Left: 0, RunFor: 600, SoC: fp(78), Kwh: fp(68.4)},
	}
	open, done := run(rs)
	if open != nil || len(done) != 1 {
		t.Fatalf("one finished start: %v %v", open, done)
	}
	s := done[0]
	if s.Start != 1180 || s.End != 1780 || *s.StartSoC != 80 || *s.EndSoC != 78 || *s.TempC != -5 || s.Drove {
		t.Fatalf("times and battery: %+v", s)
	}
	if u := s.UsedKwh(); u == nil || *u < 1.59 || *u > 1.61 {
		t.Fatalf("used: %v", u)
	}
}

func TestDrivingOffEndsItAndARestartIsCounted(t *testing.T) {
	rs := []Reading{
		{TS: 1000, Left: 590, RunFor: 600, SoC: fp(60)},
		{TS: 1500, Left: 90, RunFor: 600, SoC: fp(59)},
		{TS: 1580, Left: 600, RunFor: 600, SoC: fp(59)}, // started again
		{TS: 1700, Left: 480, RunFor: 600, SoC: fp(58)},
		{TS: 1800, Driving: true, SoC: fp(58)},
	}
	open, done := run(rs)
	if open != nil || len(done) != 1 || done[0].Restarts != 1 || !done[0].Drove || done[0].End != 1800 {
		t.Fatalf("driven off after a restart: %v %+v", open, done)
	}
}

func TestAGapEndsItAndAStillRunningOneStaysOpen(t *testing.T) {
	open, done := run([]Reading{{TS: 1000, Left: 600}, {TS: 1000 + maxGap + 10, Left: 300}})
	if len(done) != 1 || done[0].End != 1600 || open == nil || open.Start != 1000+maxGap+10 {
		t.Fatalf("a long gap starts a new one: %+v %+v", done, open)
	}
}
