package server

import (
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheFedExReportCoversAllDrivesWhenAllCategoriesIsChosen(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	start := int64(1773500000)
	end := start + 1800
	tr := &store.Trip{StartTS: start, Category: "personal", CategorySource: "manual"}
	_ = srv.S.CreateTrip(tr)
	tr.EndTS, tr.DistanceKm, tr.Open = &end, 16.0934, false
	_ = srv.S.UpdateTrip(tr)
	day := time.Unix(start, 0).In(srv.defaultLoc()).Format("2006-01-02")
	url := ts.URL + "/api/reports/trips.csv?template=fedex&from=" + day + "&to=" + day
	if _, body := do(t, admin, "GET", url, "", nil); strings.Contains(body, "Total Mileage Recorded:,,,,10") {
		t.Fatalf("with no choice the mileage log is work trips only: %q", body)
	}
	for _, c := range []string{"&categories=all", "&categories=personal"} {
		if _, body := do(t, admin, "GET", url+c, "", nil); !strings.Contains(body, "Total Mileage Recorded:,,,,10") {
			t.Fatalf("%s should include the personal drive: %q", c, body)
		}
	}
}
