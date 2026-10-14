package server

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

const exportReadme = `CarTrak export
==============
trips.csv     every recorded trip: times (UTC and local), distance in km, battery, energy used, named places
charging.csv  every charging session: energy added, the charger's own kWh and efficiency where entered, cost
places.csv    the places you named, with a centre point and radius
service.csv   your service reminders
Distances are in kilometres and energy in kWh. Times ending in Z are UTC.
It contains where your car has been, so keep it somewhere private.
`

// exportAll is a zip of spreadsheets with everything CarTrak has: trips, charging, named places and service
// reminders. Trips are read a month at a time so a long history is never cut off. Admin only.
func (s *Server) exportAll(w http.ResponseWriter, r *http.Request) {
	loc := s.defaultLoc()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="cartrak-export-%s.zip"`, time.Now().In(loc).Format("2006-01-02")))
	zw := zip.NewWriter(w)
	defer zw.Close()
	add := func(name string) (*csv.Writer, func(), error) {
		f, err := zw.Create(name)
		if err != nil {
			return nil, nil, err
		}
		c := csv.NewWriter(f)
		return c, c.Flush, nil
	}
	if f, err := zw.Create("README.txt"); err == nil {
		_, _ = f.Write([]byte(exportReadme))
	}

	if f, err := zw.Create("trips.csv"); err == nil {
		header := true
		first := s.S.FirstTripTS()
		if first > 0 {
			start := time.Unix(first, 0).In(loc)
			for m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, loc); m.Before(time.Now()); m = m.AddDate(0, 1, 0) {
				ts, err := s.S.ListTrips(tripFilterRange(m.Unix(), m.AddDate(0, 1, 0).Unix()))
				if err != nil {
					break
				}
				if len(ts) > 0 || header {
					s.writeTripsCSV(f, ts, header)
					header = false
				}
			}
		}
		if header {
			s.writeTripsCSV(f, nil, true)
		}
	}
	if f, err := zw.Create("charging.csv"); err == nil {
		ss, _ := s.chargeSessions(100000)
		s.writeChargesCSV(f, ss)
	}
	if c, flush, err := add("places.csv"); err == nil {
		_ = c.Write([]string{"name", "latitude", "longitude", "radius_m"})
		if ps, err := s.S.ListNamedPlaces(); err == nil {
			for _, p := range ps {
				_ = c.Write([]string{csvSafe(p.Name), strconv.FormatFloat(p.Lat, 'f', 6, 64), strconv.FormatFloat(p.Lon, 'f', 6, 64), strconv.FormatFloat(p.RadiusM, 'f', 0, 64)})
			}
		}
		flush()
	}
	if c, flush, err := add("service.csv"); err == nil {
		_ = c.Write([]string{"name", "every_km", "every_days", "last_done_odometer_km", "last_done"})
		if ms, err := s.S.ListMaintenance(); err == nil {
			for _, m := range ms {
				every := ""
				if m.EveryDays != nil {
					every = strconv.Itoa(*m.EveryDays)
				}
				last := ""
				if m.LastTS > 0 {
					last = time.Unix(m.LastTS, 0).In(loc).Format("2006-01-02")
				}
				_ = c.Write([]string{csvSafe(m.Name), fl(m.EveryKm), every, fl(m.LastOdoKm), last})
			}
		}
		flush()
	}
}

// tripFilterRange is every trip that started in [from, to), at most 1000 (a month never has that many).
func tripFilterRange(from, to int64) store.TripFilter {
	return store.TripFilter{From: from, To: to, Limit: 1000}
}
