package server

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

func TestTheFullExportIsAZipOfSpreadsheetsForAdminsOnly(t *testing.T) {
	ts, srv := healthServer(t, fakeProv{}, false, false)
	admin := adminClient(t, ts.URL)
	for i := 0; i < 3; i++ {
		start := time.Now().AddDate(0, -i*2, 0)
		end := start.Add(20 * time.Minute).Unix()
		tr := &store.Trip{StartTS: start.Unix(), EndTS: &end, DistanceKm: float64(10 + i), Category: "work", CategorySource: "default", Notes: "=SUM(A1)"}
		if err := srv.S.CreateTrip(tr); err != nil {
			t.Fatal(err)
		}
		tr.EndTS, tr.Open = &end, false
		if err := srv.S.UpdateTrip(tr); err != nil {
			t.Fatal(err)
		}
		_ = srv.S.SetTripCategory(tr.ID, "work", "=SUM(A1)")
	}
	code, body := do(t, admin, "GET", ts.URL+"/api/export/all.zip", "", nil)
	if code != 200 {
		t.Fatalf("%d", code)
	}
	zr, err := zip.NewReader(bytes.NewReader([]byte(body)), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(b)
	}
	for _, name := range []string{"README.txt", "trips.csv", "charging.csv", "places.csv", "service.csv"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("missing %s in %v", name, files)
		}
	}
	rows := strings.Split(strings.TrimSpace(files["trips.csv"]), "\n")
	if len(rows) != 4 || !strings.HasPrefix(rows[0], "id,start") {
		t.Fatalf("one heading and three trips across several months: %q", files["trips.csv"])
	}
	if strings.Contains(files["trips.csv"], ",=SUM") {
		t.Fatalf("a note that starts with = must not become a formula: %q", files["trips.csv"])
	}
	// A viewer cannot download it.
	do(t, admin, "POST", ts.URL+"/api/users", `{"name":"fam","password":"viewer-password-1","role":"viewer"}`, xh)
	vj, _ := cookieJar()
	v := &http.Client{Jar: vj}
	do(t, v, "POST", ts.URL+"/api/login", `{"name":"fam","password":"viewer-password-1"}`, nil)
	if code, _ := do(t, v, "GET", ts.URL+"/api/export/all.zip", "", nil); code != 403 {
		t.Fatalf("viewer: %d", code)
	}
}
