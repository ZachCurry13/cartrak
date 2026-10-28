package server

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fieldwatch"
)

const trackReadme = `CarTrak tracking export
=======================
Everything that happened between the start and the stop of a tracking session. Times ending in Z are UTC; "local" is the time zone
set in CarTrak.

debug-report.txt     the usual Debug report for the stretch (readings, trips, charging, alerts, polling, settings, Field check)
readings-raw.jsonl   every reading the car sent, with all of its fields exactly as Ford sent them, one per line
readings.csv         the main values of each reading as columns
trips.jsonl          the trips that were going or recorded in the stretch, with everything CarTrak keeps about each
charges.jsonl        the charging sessions in the stretch
decisions.csv        what CarTrak decided without leaving another trace (a trip thrown away, a charge merged, a setting changed,
                     Fix a day, tracking started and stopped). Settings are named, never their values.
alerts.csv           the alerts sent, and for how many devices
requests.csv         every request the app made: when, what, the answer's status and how long it took (nothing in the request)
app-events.csv       the pages opened, buttons tapped (by their label) and errors on the phone
field-check.txt      the Field check answers saved at the time of the export
`

const trackReadmePrivate = "Positions and the VIN are INCLUDED in this export. Keep it private.\n"
const trackReadmeClean = "Positions, the VIN and vehicle ids were left out of the readings and records. The names of pages and buttons you tapped, which can include a place name, are kept.\n"

func clock(loc *time.Location, ts int64) (string, string) {
	t := time.Unix(ts, 0)
	return t.UTC().Format(time.RFC3339), t.In(loc).Format("2006-01-02 15:04:05")
}

func numText(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}

// trackExport: GET /api/track/export?private=1. A zip of everything in the last tracking session. Admin only. With all=1 (an
// administrator of the whole installation) it holds every car, each in a folder named by its number.
func (s *Server) trackExport(w http.ResponseWriter, r *http.Request) {
	private := r.URL.Query().Get("private") == "1"
	all := r.URL.Query().Get("all") == "1" && s.Cars != nil && siteAdmin(r)
	from, _, have := s.trackSpan()
	name := fmt.Sprintf("cartrak-tracking-%s-%s.zip", fileSlug(s.carName()), time.Unix(from, 0).In(s.defaultLoc()).Format("2006-01-02-1504"))
	if all {
		name = fmt.Sprintf("cartrak-tracking-all-cars-%s.zip", time.Now().In(s.defaultLoc()).Format("2006-01-02-1504"))
		have = true
	}
	if !have {
		fail(w, 404, "there is no tracking session yet")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	zw := zip.NewWriter(w)
	defer zw.Close()
	metric := metricReq(r)
	if !all {
		s.writeTrackZip(zw, "", private, metric, r)
		return
	}
	if f, err := zw.Create("README.txt"); err == nil {
		_, _ = f.Write([]byte("CarTrak tracking export, every car. Each car has a folder named by its number with the files described in its own README.\nThe requests and the pages, taps and errors of the phone are recorded by the car that was on screen at the time.\n"))
	}
	s.Cars.EachCar(func(id int64, c *Server) {
		c.writeTrackZip(zw, fmt.Sprintf("car-%d/", id), private, metric, r)
	})
}

// writeTrackZip writes this car's tracking files into zw, each under prefix.
func (s *Server) writeTrackZip(zw *zip.Writer, prefix string, private, metric bool, r *http.Request) {
	from, to, have := s.trackSpan()
	loc := s.defaultLoc()
	if !have {
		if f, err := zw.Create(prefix + "NO-SESSION.txt"); err == nil {
			_, _ = f.Write([]byte("tracking was never started on this car\n"))
		}
		return
	}
	file := func(name string, fill func(io.Writer)) {
		if f, err := zw.Create(prefix + name); err == nil {
			fill(f)
		}
	}
	table := func(name string, header []string, fill func(c *csv.Writer)) {
		file(name, func(f io.Writer) {
			c := csv.NewWriter(f)
			_ = c.Write(header)
			fill(c)
			c.Flush()
		})
	}
	file("README.txt", func(f io.Writer) {
		fmt.Fprint(f, trackReadme)
		fmt.Fprintf(f, "\nStretch: %s to %s (%s)\n", time.Unix(from, 0).UTC().Format(time.RFC3339), time.Unix(to, 0).UTC().Format(time.RFC3339), loc)
		if private {
			fmt.Fprint(f, trackReadmePrivate)
		} else {
			fmt.Fprint(f, trackReadmeClean)
		}
	})
	file("debug-report.txt", func(f io.Writer) {
		hours := min(336, int((time.Now().Unix()-from)/3600)+2)
		text, _, _ := s.DebugText(hours, metric)
		fmt.Fprintf(f, "TRACKING: the stretch is %s to %s. The report below also covers some time before it.\n\n%s", time.Unix(from, 0).In(loc).Format("01-02 15:04:05"), time.Unix(to, 0).In(loc).Format("01-02 15:04:05"), text)
	})
	file("readings-raw.jsonl", func(f io.Writer) {
		_ = s.S.EachExtra(from, to, func(ts int64, extra []byte) error {
			m := map[string]any{}
			if json.Unmarshal(extra, &m) != nil {
				return nil
			}
			if !private {
				for k := range m {
					if fieldwatch.Private(k) {
						delete(m, k)
					}
				}
			}
			b, _ := json.Marshal(map[string]any{"ts": ts, "fields": m})
			_, _ = f.Write(append(b, '\n'))
			return r.Context().Err()
		})
	})
	table("readings.csv", []string{"utc", "local", "odometer_km", "soc_pct", "energy_kwh", "range_km", "lat", "lon", "speed_kph", "ignition", "charge_state", "plug", "charge_kw", "outside_c", "battery12v", "tire_fl", "tire_fr", "tire_rl", "tire_rr", "locked", "fuel_pct"}, func(c *csv.Writer) {
		ss, _ := s.S.SamplesBetween(from, to)
		for _, x := range ss {
			u, l := clock(loc, x.TS)
			lat, lon := x.Lat, x.Lon
			if !private {
				lat, lon = nil, nil
			}
			locked := ""
			if x.Locked != nil {
				locked = strconv.FormatBool(*x.Locked)
			}
			_ = c.Write([]string{u, l, numText(x.OdometerKm), numText(x.SoCPct), numText(x.EnergyKwh), numText(x.RangeKm), numText(lat), numText(lon), numText(x.SpeedKph), x.Ignition, x.ChargeState, x.Plug,
				numText(x.ChargeKw), numText(x.OutsideC), numText(x.Battery12V), numText(x.Tires[0]), numText(x.Tires[1]), numText(x.Tires[2]), numText(x.Tires[3]), locked, numText(x.FuelPct)})
		}
	})
	s.trackRecords(file, from, to, private)
	table("decisions.csv", []string{"utc", "local", "kind", "note"}, func(c *csv.Writer) {
		for _, e := range s.S.RecentDecisions(from) {
			if e.TS <= to {
				u, l := clock(loc, e.TS)
				_ = c.Write([]string{u, l, e.Kind, safeCell(e.Note)})
			}
		}
	})
	table("alerts.csv", []string{"utc", "local", "kind", "title", "body", "wanted", "sent", "quiet", "devices"}, func(c *csv.Writer) {
		as, _ := s.S.RecentAlerts(300)
		for i := len(as) - 1; i >= 0; i-- {
			if a := as[i]; a.TS >= from && a.TS <= to {
				u, l := clock(loc, a.TS)
				_ = c.Write([]string{u, l, a.Kind, safeCell(a.Title), safeCell(a.Body), strconv.Itoa(a.Wanted), strconv.Itoa(a.Sent), strconv.Itoa(a.Quiet), strconv.Itoa(a.Devices)})
			}
		}
	})
	es, _ := s.S.TrackEventsBetween(from, to)
	table("requests.csv", []string{"utc", "local", "request", "status", "ms"}, func(c *csv.Writer) {
		for _, e := range es {
			if e.Kind == "request" {
				u, l := clock(loc, e.TS)
				_ = c.Write([]string{u, l, safeCell(e.A), e.B, strconv.FormatInt(e.N, 10)})
			}
		}
	})
	table("app-events.csv", []string{"utc", "local", "what", "detail", "page"}, func(c *csv.Writer) {
		for _, e := range es {
			if e.Kind != "request" {
				u, l := clock(loc, e.TS)
				_ = c.Write([]string{u, l, e.Kind, safeCell(e.A), safeCell(e.B)})
			}
		}
	})
	file("field-check.txt", func(f io.Writer) {
		t := s.fieldCheckText()
		if t == "" {
			t = "no Field check answers were saved\n"
		}
		_, _ = f.Write([]byte(t))
	})
}
