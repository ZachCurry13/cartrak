package server

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func zipFiles(t *testing.T, body []byte) map[string][][]string {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	out := map[string][][]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		if strings.HasSuffix(f.Name, ".csv") {
			rows, _ := csv.NewReader(bytes.NewReader(b)).ReadAll()
			out[f.Name] = rows
		} else {
			out[f.Name] = [][]string{{string(b)}}
		}
	}
	return out
}

func TestTheCarFieldsExportHasEveryFieldAndEveryChange(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	t0 := time.Now().Add(-3 * time.Hour).Unix()
	add := func(i int64, speed, fuel string, door bool) {
		x := fmt.Sprintf(`{"garage.modelName":"Escape","garage.vin":"1FMXYZ","telemetry.metrics.speed.value":%s,"telemetry.metrics.speed.updateTime":"2026-10-21T10:%02d:00Z",`+
			`"telemetry.metrics.fuelLevel.value":%s,"telemetry.metrics.position.value.location.lat":44.7`, speed, i, fuel)
		if door {
			x += `,"telemetry.metrics.doorStatus[0].value":"AJAR"`
		}
		x += `}`
		srv.S.DB.Exec(`INSERT INTO samples(ts,extra) VALUES(?,?)`, t0+i*60, x)
	}
	add(0, "0", "50.5", false)
	add(1, "0", "50.5", false) // nothing changed: not repeated
	add(2, "30", "50.5", false)
	add(3, "30", "49.9", true) // a door appears
	add(4, "0", "49.9", true)

	code, body := do(t, admin, "GET", ts.URL+"/api/export/fields.zip?days=1", "", nil)
	if code != 200 {
		t.Fatalf("export: %d %.200s", code, body)
	}
	files := zipFiles(t, []byte(body))
	ch := files["changes.csv"]
	head := strings.Join(ch[0], ",")
	if !strings.HasPrefix(head, "time_utc,time_local,") || !strings.Contains(head, "telemetry.metrics.speed.value") || !strings.Contains(head, "telemetry.metrics.fuelLevel.value") ||
		!strings.Contains(head, "telemetry.metrics.doorStatus[0].value") || strings.Contains(head, "updateTime") || strings.Contains(head, "modelName") {
		t.Fatalf("the columns are the fields that changed: %s", head)
	}
	if len(ch) != 5 { // the header and four moments: 0, 2, 3, 4 (the second reading repeated the first)
		t.Fatalf("a row each time something changed: %d rows: %v", len(ch), ch)
	}
	un := fmt.Sprint(files["unchanged.csv"])
	if !strings.Contains(un, "garage.modelName Escape") || strings.Contains(un, "vin") || strings.Contains(un, "location") {
		t.Fatalf("the constant fields, without the VIN or the position: %s", un)
	}
	if latest := fmt.Sprint(files["latest-reading.csv"]); !strings.Contains(latest, "updateTime") || strings.Contains(latest, "1FMXYZ") || strings.Contains(latest, "44.7") {
		t.Fatalf("the newest reading in full, minus the private fields: %s", latest)
	}

	_, body = do(t, admin, "GET", ts.URL+"/api/export/fields.zip?days=1&private=1", "", nil)
	if un := fmt.Sprint(zipFiles(t, []byte(body))["unchanged.csv"]); !strings.Contains(un, "1FMXYZ") || !strings.Contains(un, "location") {
		t.Fatalf("asked for them: %s", un)
	}
	if code, _ := do(t, &http.Client{}, "GET", ts.URL+"/api/export/fields.zip", "", nil); code == 200 {
		t.Fatal("signed-out callers get nothing")
	}
	srv.S.DB.Exec(`DELETE FROM samples`)
	if code, _ := do(t, admin, "GET", ts.URL+"/api/export/fields.zip?days=1", "", nil); code != 404 {
		t.Fatal("nothing to export says so")
	}
}

func TestFileSlugIsSafeForFileNames(t *testing.T) {
	for in, want := range map[string]string{"My Mach-E": "my-mach-e", "  ": "car", "Escape (hybrid)!": "escape-hybrid"} {
		if got := fileSlug(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}
