package server

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	// poller.lastOk is when CarTrak last got an answer from Ford; carLastReport is when the car itself last had
	// something new (Ford's own update times). They differ when the car is asleep or out of signal.
	ok(w, map[string]any{"poller": s.P.Status(), "canLink": s.Link != nil, "configured": s.Cfg == nil || s.Cfg.Configured(), "carLastReport": s.S.CarLastReport()})
}

func (s *Server) latest(w http.ResponseWriter, r *http.Request) {
	x, err := s.S.LatestSample()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	if x == nil {
		ok(w, nil) // null before the first reading
		return
	}
	// ?summary=1 is the small data-saver version. The ETag lets clients get a tiny
	// "304 unchanged" instead of the whole reading when nothing is new.
	sum := r.URL.Query().Get("summary") == "1"
	etag := fmt.Sprintf(`W/"%d-%t"`, x.TS, sum)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if sum {
		ok(w, summarize(x))
		return
	}
	ok(w, x)
}

func tripFilter(r *http.Request) store.TripFilter {
	q := r.URL.Query()
	f := store.TripFilter{Category: q.Get("category")}
	f.From, _ = strconv.ParseInt(q.Get("from"), 10, 64)
	f.To, _ = strconv.ParseInt(q.Get("to"), 10, 64)
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	return f
}

func (s *Server) listTrips(w http.ResponseWriter, r *http.Request) {
	ts, err := s.S.ListTrips(tripFilter(r))
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, s.tripsWithEnergy(ts))
}

func (s *Server) tripDetail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, err := s.S.GetTrip(id)
	if err != nil {
		fail(w, 404, "no such trip")
		return
	}
	end := time.Now().Unix()
	if t.EndTS != nil {
		end = *t.EndTS
	}
	if r.URL.Query().Get("samples") == "0" { // data saver: skip the (large) list of route points
		ok(w, map[string]any{"trip": s.placeNames(withEnergy(t, s.capacityKwh())), "samples": []any{}, "phone": []any{}, "samplesSkipped": true})
		return
	}
	ss, _ := s.S.TrackBetween(t.StartTS, end)
	phone := s.phoneRoute(t.StartTS, end)
	var est *estimateOut
	if len(phone) < 2 {
		est = s.estimateFor(t, true)
	}
	ok(w, map[string]any{"trip": s.placeNames(withEnergy(t, s.capacityKwh())), "samples": ss, "phone": phone, "estimate": est})
}

func (s *Server) patchTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var b struct{ Category, Notes string }
	if decode(r, &b) != nil || !s.S.CategoryExists(b.Category) {
		fail(w, 400, "choose one of your trip categories")
		return
	}
	if err := s.S.SetTripCategory(id, b.Category, b.Notes); err != nil {
		fail(w, 500, "db error")
		return
	}
	t, _ := s.S.GetTrip(id)
	ok(w, t)
}

// summary totals distance and trip counts per category for a date range.
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	f.Category, f.Limit = "", 1000
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	type agg struct {
		Trips      int     `json:"trips"`
		DistanceKm float64 `json:"distanceKm"`
		KWh        float64 `json:"kwh"`
		EnergyKm   float64 `json:"energyKm"` // distance of the trips that have an energy figure, for efficiency
	}
	capacity := s.capacityKwh()
	out := map[string]*agg{"work": {}, "personal": {}}
	for _, t := range ts {
		if t.Open {
			continue
		}
		a := out[t.Category]
		if a == nil {
			a = &agg{}
			out[t.Category] = a
		}
		a.Trips++
		a.DistanceKm += t.DistanceKm
		if e, ok := trips.EnergyUsed(t, capacity); ok {
			a.KWh += e.Kwh
			a.EnergyKm += t.DistanceKm
		}
	}
	ok(w, out)
}

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	f := tripFilter(r)
	f.Limit = 1000
	ts, err := s.S.ListTrips(f)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="cartrak-trips.csv"`)
	s.writeTripsCSV(w, ts, true)
}

// writeTripsCSV writes trips as a spreadsheet; header says whether to start with the heading row.
func (s *Server) writeTripsCSV(w io.Writer, ts []*store.Trip, header bool) {
	c := csv.NewWriter(w)
	if header {
		_ = c.Write([]string{"id", "start", "end", "start_local", "timezone", "category", "source", "distance_km", "start_soc", "end_soc", "max_speed_kph", "notes", "energy_kwh", "energy_estimated", "start_place", "end_place", "distance_mi", "max_speed_mph"})
	}
	for _, o := range s.tripsWithEnergy(ts) {
		t := o.Trip
		end := ""
		if t.EndTS != nil {
			end = time.Unix(*t.EndTS, 0).UTC().Format(time.RFC3339)
		}
		_ = c.Write([]string{strconv.FormatInt(t.ID, 10), time.Unix(t.StartTS, 0).UTC().Format(time.RFC3339), end,
			localStart(t), t.TZ, t.Category, t.CategorySource, fmt.Sprintf("%.2f", t.DistanceKm), fl(t.StartSoC), fl(t.EndSoC),
			fmt.Sprintf("%.0f", t.MaxSpeedKph), csvSafe(t.Notes), fl2(o.EnergyKwh), strconv.FormatBool(o.EnergyEstimated), csvSafe(o.StartPlace), csvSafe(o.EndPlace),
			fmt.Sprintf("%.2f", t.DistanceKm/kmPerMile), fmt.Sprintf("%.0f", t.MaxSpeedKph/kmPerMile)})
	}
	c.Flush()
}

// fl2 is a number with two decimals, or empty when not known.
func fl2(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *p)
}

func fl(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.1f", *p)
}

// csvSafe stops spreadsheet formula injection from user-entered notes.
func csvSafe(s string) string {
	if s != "" && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}
