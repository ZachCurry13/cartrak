package routeest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDrivePaceSplitsTheDriveIntoHighwayAndTheRest(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.RawQuery
		// 60 km of highway at 100 km/h (27.78 m/s) and 10 km of town at 40 km/h (11.11 m/s), timed as the service would
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":70000,"duration":4950,"legs":[{"annotation":{"distance":[30000,30000,10000],"speed":[27.78,27.78,11.11]}}]}]}`))
	}))
	defer srv.Close()
	p, err := (&Client{BaseURL: srv.URL}).DrivePace(context.Background(), []Point{{45, -93}, {45.5, -92}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asked, "annotations=speed,distance") {
		t.Errorf("it must ask for the speeds: %q", asked)
	}
	if p.Km != 70 || p.Minutes != 82.5 {
		t.Errorf("the whole drive is the service's own figure: %+v", p)
	}
	if p.HighwayKm < 59.9 || p.HighwayKm > 60.1 || p.HighwayMin < 35.9 || p.HighwayMin > 36.1 {
		t.Errorf("60 km at 100 km/h is 36 minutes: %+v", p)
	}
}
