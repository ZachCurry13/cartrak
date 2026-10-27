package server

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

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
