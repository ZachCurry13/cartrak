package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/maint"
	"github.com/ZachCurry13/cartrak/internal/store"
)

func (s *Server) odometerKm() *float64 {
	if x, _ := s.S.LatestSample(); x != nil {
		return x.OdometerKm
	}
	return nil
}

// listMaintenance is every service reminder with how much is left of it.
func (s *Server) listMaintenance(w http.ResponseWriter, r *http.Request) {
	items, err := s.S.ListMaintenance()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	odo, now := s.odometerKm(), time.Now().Unix()
	out := make([]maint.Status, 0, len(items))
	for _, m := range items {
		out = append(out, maint.Check(m, odo, now))
	}
	ok(w, map[string]any{"items": out, "odometerKm": odo})
}

type maintIn struct {
	Name      *string  `json:"name"`
	EveryKm   *float64 `json:"everyKm"`
	EveryDays *int     `json:"everyDays"`
	ClearKm   bool     `json:"clearKm"`
	ClearDays bool     `json:"clearDays"`
}

func (in *maintIn) valid() string {
	if in.Name != nil && (strings.TrimSpace(*in.Name) == "" || len(*in.Name) > 60) {
		return "give the reminder a short name"
	}
	if in.EveryKm != nil && (*in.EveryKm < 100 || *in.EveryKm > 200000) {
		return "the distance must be between 100 km and 200,000 km"
	}
	if in.EveryDays != nil && (*in.EveryDays < 7 || *in.EveryDays > 3650) {
		return "the time must be between 7 days and 10 years"
	}
	return ""
}

// addMaintenance creates a reminder that starts counting now, from the car's current odometer. Admin only.
func (s *Server) addMaintenance(w http.ResponseWriter, r *http.Request) {
	var in maintIn
	if decode(r, &in) != nil || in.Name == nil || (in.EveryKm == nil && in.EveryDays == nil) {
		fail(w, 400, "give a name and a distance, a time, or both")
		return
	}
	if msg := in.valid(); msg != "" {
		fail(w, 400, msg)
		return
	}
	m := &store.Maintenance{Name: strings.TrimSpace(*in.Name), EveryKm: in.EveryKm, EveryDays: in.EveryDays, LastOdoKm: s.odometerKm(), LastTS: time.Now().Unix()}
	if err := s.S.SaveMaintenance(m); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"id": m.ID})
}

func (s *Server) patchMaintenance(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in maintIn
	m, err := s.S.GetMaintenance(id)
	if decode(r, &in) != nil || err != nil || m == nil {
		fail(w, 404, "no such reminder")
		return
	}
	if msg := in.valid(); msg != "" {
		fail(w, 400, msg)
		return
	}
	if in.Name != nil {
		m.Name = strings.TrimSpace(*in.Name)
	}
	if in.EveryKm != nil || in.ClearKm {
		m.EveryKm = in.EveryKm
	}
	if in.EveryDays != nil || in.ClearDays {
		m.EveryDays = in.EveryDays
	}
	if m.EveryKm == nil && m.EveryDays == nil {
		fail(w, 400, "keep a distance or a time")
		return
	}
	m.NotifiedTS = 0
	if err := s.S.SaveMaintenance(m); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "saved")
}

// maintenanceDone records that the work was done now.
func (s *Server) maintenanceDone(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.MarkMaintenanceDone(id, s.odometerKm()); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "done")
}

func (s *Server) deleteMaintenance(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteMaintenance(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}
