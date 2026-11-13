package nws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sample = `{"features":[
 {"properties":{"id":"a1","event":"Winter Weather Advisory","severity":"Moderate","headline":"Snow 2 to 4 inches","areaDesc":"Hennepin","status":"Actual","messageType":"Alert","onset":"2026-11-16T06:00:00-06:00","ends":"2026-11-16T18:00:00-06:00"}},
 {"properties":{"id":"a2","event":"Winter Storm Warning","severity":"Severe","headline":"Heavy snow","areaDesc":"Hennepin","status":"Actual","messageType":"Update","onset":"2026-11-16T12:00:00-06:00","expires":"2026-11-17T06:00:00-06:00"}},
 {"properties":{"id":"t","event":"Test Message","severity":"Unknown","status":"Test","messageType":"Alert"}},
 {"properties":{"id":"s","event":"Special Weather Statement","severity":"Moderate","status":"Actual","messageType":"Alert"}},
 {"properties":{"id":"h","event":"Heat Advisory","severity":"Moderate","status":"Actual","messageType":"Alert"}}]}`

func TestActiveSendsARoundedPointNamesItselfAndSortsBySeverity(t *testing.T) {
	var q, ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, ua = r.URL.RawQuery, r.Header.Get("User-Agent")
		w.Write([]byte(sample))
	}))
	defer srv.Close()
	got, err := Active(context.Background(), nil, srv.URL, 44.98765, -93.26543)
	if err != nil {
		t.Fatal(err)
	}
	if q != "point=45.0,-93.3" || ua == "" {
		t.Fatalf("a rounded point and a name: %q %q", q, ua)
	}
	if len(got) != 4 || got[0].ID != "a2" || got[0].Ends == 0 || got[1].Onset == 0 {
		t.Fatalf("tests left out, the most severe first, times read: %+v", got)
	}
	var drive []string
	for _, a := range got {
		if ForDriving(a) {
			drive = append(drive, a.ID)
		}
	}
	if len(drive) != 2 || drive[0] != "a2" || drive[1] != "a1" {
		t.Fatalf("only the road warnings are worth an alert: %v", drive)
	}
}

func TestOutsideTheUSSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
	defer srv.Close()
	if _, err := Active(context.Background(), nil, srv.URL, 48.85, 2.35); err == nil || err.Error() != "the weather service only covers the United States" {
		t.Fatalf("%v", err)
	}
}
