package batteryhealth

import (
	"testing"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func series(days int, start, end float64) []store.Day {
	out := make([]store.Day, days)
	for i := range out {
		v := start + (end-start)*float64(i)/float64(days-1)
		out[i] = store.Day{TS: int64(i) * 86400, Avg: v, Min: v, Max: v, N: 1}
	}
	return out
}

func TestVerdictsFollowTheSizeOfALastingDrop(t *testing.T) {
	cases := []struct {
		end  float64
		want string
	}{{88, "steady"}, {87, "steady"}, {84, "gentle"}, {78, "watch"}}
	for _, c := range cases {
		r := Build(series(200, 88, c.end))
		if !r.Ready || r.Verdict != c.want {
			t.Errorf("88 to %v: ready=%v verdict=%q change=%.1f", c.end, r.Ready, r.Verdict, r.ChangePct)
		}
	}
}

func TestTooLittleHistoryGivesNoVerdictButStillListsMonths(t *testing.T) {
	r := Build(series(20, 88, 80))
	if r.Ready || r.Verdict != "" || len(r.Months) == 0 {
		t.Fatalf("%+v", r)
	}
	if r := Build(nil); r.Ready || len(r.Months) != 0 {
		t.Fatalf("%+v", r)
	}
	if r := Build(series(50, 88, 80)); r.Ready {
		t.Fatal("under 60 days apart is not a trend")
	}
}
