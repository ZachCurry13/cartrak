package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// The service history: visits found at service places (waiting for an answer, or answered) and ones logged by hand. Saying what
// was done resets the service reminders of the same name, and "due again in" creates or changes them.

type dueIn struct {
	Item      string   `json:"item"`
	EveryKm   *float64 `json:"everyKm"`
	EveryDays *int     `json:"everyDays"`
}

type serviceIn struct {
	Status    string   `json:"status"` // logged or dismissed
	PlaceName string   `json:"placeName"`
	Start     int64    `json:"start"`
	OdoKm     *float64 `json:"odoKm"`
	Items     []string `json:"items"`
	Notes     string   `json:"notes"`
	Cost      *float64 `json:"cost"`
	Due       []dueIn  `json:"due"`
}

func (in *serviceIn) check() string {
	if len(in.Items) > 15 || len(in.Notes) > 500 || len(in.PlaceName) > 60 || len(in.Due) > 15 {
		return "that is too long"
	}
	for _, it := range in.Items {
		if strings.TrimSpace(it) == "" || len(it) > 60 {
			return "each item needs a short name"
		}
	}
	if in.Cost != nil && (*in.Cost < 0 || *in.Cost > 100000) {
		return "the cost must be between 0 and 100,000"
	}
	for _, d := range in.Due {
		if (d.EveryKm != nil && (*d.EveryKm <= 0 || *d.EveryKm > 500000)) || (d.EveryDays != nil && (*d.EveryDays <= 0 || *d.EveryDays > 3650)) {
			return "due again in must be a sensible distance or time"
		}
	}
	return ""
}

func (s *Server) serviceHistory(w http.ResponseWriter, r *http.Request) {
	vs, err := s.S.Visits(time.Now().AddDate(-5, 0, 0).Unix(), 0, "")
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ms, _ := s.S.ListMaintenance()
	ok(w, map[string]any{"visits": vs, "reminders": ms, "currency": s.currency()})
}

// applyDue resets the reminders matching what was done, and creates or changes the ones given a "due again in".
func (s *Server) applyDue(items []string, due []dueIn, odo *float64, at int64) {
	ms, _ := s.S.ListMaintenance()
	find := func(name string) *store.Maintenance {
		for _, m := range ms {
			if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(name)) {
				return m
			}
		}
		return nil
	}
	given := map[string]bool{}
	for _, d := range due {
		given[strings.ToLower(d.Item)] = true
		m := find(d.Item)
		if m == nil {
			if d.EveryKm == nil && d.EveryDays == nil {
				continue
			}
			m = &store.Maintenance{Name: strings.TrimSpace(d.Item)}
			ms = append(ms, m)
		}
		if d.EveryKm != nil || d.EveryDays != nil {
			m.EveryKm, m.EveryDays = d.EveryKm, d.EveryDays
		}
		m.LastOdoKm, m.LastTS, m.NotifiedTS = odo, at, 0
		_ = s.S.SaveMaintenance(m)
	}
	for _, it := range items {
		if m := find(it); m != nil && !given[strings.ToLower(it)] {
			m.LastOdoKm, m.LastTS, m.NotifiedTS = odo, at, 0
			_ = s.S.SaveMaintenance(m)
		}
	}
}

// answerVisit says what happened on a visit CarTrak found: logged with what was done, or dismissed. PATCH /api/service/{id}
func (s *Server) answerVisit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	v, err := s.S.GetVisit(id)
	var in serviceIn
	if err != nil || v == nil || decode(r, &in) != nil {
		fail(w, 404, "no such visit")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	if in.Status == "dismissed" {
		v.Status = "dismissed"
	} else {
		v.Status, v.Items, v.Notes, v.Cost = "logged", in.Items, strings.TrimSpace(in.Notes), in.Cost
		if in.OdoKm != nil {
			v.OdoKm = in.OdoKm
		}
		s.applyDue(in.Items, in.Due, v.OdoKm, v.End)
	}
	if err := s.S.SaveVisit(v); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, v)
}

// logVisit adds a service visit by hand, for one CarTrak did not see. POST /api/service
func (s *Server) logVisit(w http.ResponseWriter, r *http.Request) {
	var in serviceIn
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	if len(in.Items) == 0 {
		fail(w, 400, "say what was done")
		return
	}
	at := in.Start
	if at <= 0 || at > time.Now().Unix()+86400 {
		at = time.Now().Unix()
	}
	odo := in.OdoKm
	if odo == nil {
		if x, err := s.S.LatestSample(); err == nil && x != nil {
			odo = x.OdometerKm
		}
	}
	v := &store.ServiceVisit{PlaceName: strings.TrimSpace(in.PlaceName), Start: at, End: at, OdoKm: odo, Status: "logged", Items: in.Items, Notes: strings.TrimSpace(in.Notes), Cost: in.Cost}
	if err := s.S.SaveVisit(v); err != nil {
		fail(w, 500, "db error")
		return
	}
	s.applyDue(in.Items, in.Due, odo, at)
	ok(w, v)
}

func (s *Server) deleteVisit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteVisit(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}

// exportServiceCSV is the service history as a spreadsheet: one row a visit, with what was done and what it cost.
// GET /api/export/service.csv
func (s *Server) exportServiceCSV(w http.ResponseWriter, r *http.Request) {
	vs, err := s.S.Visits(0, 0, "logged")
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="cartrak-service.csv"`)
	c := csv.NewWriter(w)
	_ = c.Write([]string{"date", "place", "work_done", "cost", "odometer_mi", "odometer_km", "notes"})
	loc := s.defaultLoc()
	for _, v := range vs {
		mi, km := "", ""
		if v.OdoKm != nil {
			mi, km = fmt.Sprintf("%.0f", *v.OdoKm*0.621371), fmt.Sprintf("%.0f", *v.OdoKm)
		}
		_ = c.Write([]string{time.Unix(v.Start, 0).In(loc).Format("2006-01-02"), v.PlaceName, strings.Join(v.Items, "; "), fl2(v.Cost), mi, km, v.Notes})
	}
	c.Flush()
}
