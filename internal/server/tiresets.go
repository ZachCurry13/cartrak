package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tirecompare"
)

// tireSetList lists the tire sets and the one on now. GET /api/tire-sets
func (s *Server) tireSetList(w http.ResponseWriter, r *http.Request) {
	sets, err := s.S.TireSets()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	out := map[string]any{"sets": sets, "kinds": store.TireKinds}
	if cur := store.TireAt(sets, time.Now().Unix()); cur != nil {
		out["current"] = cur
	}
	ok(w, out)
}

func cleanTire(in store.TireSet) (store.TireSet, string) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 60 {
		return in, "give the set a name of up to 60 characters"
	}
	if !store.ValidTireKind(in.Kind) {
		return in, "choose winter, summer, all-season or other"
	}
	if in.FromTS <= 0 || in.FromTS > time.Now().Add(24*time.Hour).Unix() {
		return in, "choose the date they went on (not in the future)"
	}
	if len(in.Note) > 300 {
		in.Note = in.Note[:300]
	}
	return in, ""
}

// tireSetAdd records a set going on. POST /api/tire-sets {name, kind, fromTs, note}. Admin only.
func (s *Server) tireSetAdd(w http.ResponseWriter, r *http.Request) {
	var in store.TireSet
	if decode(r, &in) != nil {
		fail(w, 400, "that did not read")
		return
	}
	t, why := cleanTire(in)
	if why != "" {
		fail(w, 400, why)
		return
	}
	id, err := s.S.AddTireSet(t)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	s.S.LogDecision("tires", "tire set added: "+t.Name+" ("+t.Kind+")")
	ok(w, map[string]any{"id": id})
}

// tireSetChange edits a set. PATCH /api/tire-sets/{id}. Admin only.
func (s *Server) tireSetChange(w http.ResponseWriter, r *http.Request) {
	var in store.TireSet
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if decode(r, &in) != nil {
		fail(w, 400, "that did not read")
		return
	}
	in.ID = id
	t, why := cleanTire(in)
	if why != "" {
		fail(w, 400, why)
		return
	}
	if err := s.S.UpdateTireSet(t); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "saved")
}

// tireSetRemove deletes a set. DELETE /api/tire-sets/{id}. Admin only.
func (s *Server) tireSetRemove(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteTireSet(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "removed")
}

// tireTrips are the finished trips of the last year as the comparison needs them: energy, outside temperature, average speed, whether the
// road was wet, and whether the car was on winter tires.
func (s *Server) tireTrips(sets []store.TireSet) ([]tirecompare.Trip, error) {
	from := time.Now().AddDate(0, 0, -366).Unix()
	ts, err := s.S.EnergyTrips(from)
	if err != nil {
		return nil, err
	}
	temps, err := s.S.TripTemps(from)
	if err != nil {
		return nil, err
	}
	out := make([]tirecompare.Trip, 0, len(ts))
	for _, t := range ts {
		if t.EndTS == nil || t.StartKwh == nil || t.EndKwh == nil || *t.EndTS <= t.StartTS {
			continue
		}
		set := store.TireAt(sets, t.StartTS)
		if set == nil {
			continue // before the first set was logged: not known
		}
		x := tirecompare.Trip{ID: t.ID, Km: t.DistanceKm, Kwh: *t.StartKwh - *t.EndKwh, Winter: set.Kind == "winter",
			AvgKph: t.DistanceKm / (float64(*t.EndTS-t.StartTS) / 3600)}
		if c, found := temps[t.ID]; found {
			x.TempC, x.HasT = c, true
		}
		if t.StartLat != nil && t.StartLon != nil {
			if mm, known := s.rainDuring(*t.StartLat, *t.StartLon, t.StartTS, *t.EndTS); known && mm >= wetFromMm {
				x.Wet = true
			}
		}
		out = append(out, x)
	}
	return out, nil
}

// tireSetCompare says whether winter tires change this car's range, at the same temperature. GET /api/tire-sets/compare
func (s *Server) tireSetCompare(w http.ResponseWriter, r *http.Request) {
	sets, err := s.S.TireSets()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	trips, err := s.tireTrips(sets)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	res := tirecompare.Compare(trips)
	hasWinter, hasOther := false, false
	for _, t := range sets {
		if t.Kind == "winter" {
			hasWinter = true
		} else {
			hasOther = true
		}
	}
	out := map[string]any{"result": res, "hasWinter": hasWinter, "hasOther": hasOther, "capacityKwh": s.capacityKwh()}
	ok(w, out)
}
