package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/report"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// templateConfig is the layout of a custom report, kept as JSON in the report_templates table.
type templateConfig struct {
	Columns []report.Pick  `json:"columns"`
	Fixed   []report.Fixed `json:"fixed"`
	Summary bool           `json:"summary"` // the summary block and the totals sheet
	Title   string         `json:"title"`
}

func (c *templateConfig) check() string {
	if len(c.Columns) == 0 || len(c.Columns) > 40 {
		return "choose between 1 and 40 columns"
	}
	for i := range c.Columns {
		if !report.ValidKey(c.Columns[i].Key) {
			return "unknown column " + c.Columns[i].Key
		}
		if r := []rune(c.Columns[i].Label); len(r) > 60 {
			c.Columns[i].Label = string(r[:60])
		}
	}
	if len(c.Fixed) > 10 {
		return "at most 10 fixed columns"
	}
	for i := range c.Fixed {
		if strings.TrimSpace(c.Fixed[i].Label) == "" {
			return "every fixed column needs a heading"
		}
		if r := []rune(c.Fixed[i].Label); len(r) > 60 {
			c.Fixed[i].Label = string(r[:60])
		}
		if r := []rune(c.Fixed[i].Value); len(r) > 200 {
			c.Fixed[i].Value = string(r[:200])
		}
	}
	if r := []rune(c.Title); len(r) > 80 {
		c.Title = string(r[:80])
	}
	return ""
}

// reportOptions: GET /api/reports/options. What the report builder offers: the columns, the categories and the saved custom reports.
func (s *Server) reportOptions(w http.ResponseWriter, r *http.Request) {
	cats, _ := s.S.ListCategories()
	ts, _ := s.S.ListReportTemplates()
	type tpl struct {
		ID     int64          `json:"id"`
		Name   string         `json:"name"`
		Config templateConfig `json:"config"`
	}
	out := make([]tpl, 0, len(ts))
	for _, t := range ts {
		var c templateConfig
		_ = json.Unmarshal([]byte(t.Config), &c)
		out = append(out, tpl{t.ID, t.Name, c})
	}
	ok(w, map[string]any{"columns": report.Columns, "categories": cats, "templates": out})
}

// saveReportTemplate: POST /api/reports/templates {id?, name, config}. Admin only.
func (s *Server) saveReportTemplate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     int64          `json:"id"`
		Name   string         `json:"name"`
		Config templateConfig `json:"config"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := in.Config.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	b, _ := json.Marshal(in.Config)
	t := &store.ReportTemplate{ID: in.ID, Name: in.Name, Config: string(b)}
	if err := s.S.SaveReportTemplate(t); err != nil {
		fail(w, 400, err.Error())
		return
	}
	ok(w, map[string]int64{"id": t.ID})
}

// deleteReportTemplate: DELETE /api/reports/templates/{id}. Admin only.
func (s *Server) deleteReportTemplate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.DeleteReportTemplate(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "deleted")
}

// reportTrips turns stored trips into report rows: names for the places, the time zone each trip started in, and whether a phone
// recorded the route. Only finished trips are listed.
func (s *Server) reportTrips(ts []*store.Trip, loc *time.Location) []report.Trip {
	out := make([]report.Trip, 0, len(ts))
	for _, o := range s.tripsWithEnergy(ts) {
		t := o.Trip
		if t.Open || t.EndTS == nil {
			continue
		}
		l := loc
		if t.TZ != "" {
			if x, err := time.LoadLocation(t.TZ); err == nil {
				l = x
			}
		}
		name := func(label string, lat, lon *float64) string {
			if label != "" {
				return label
			}
			if lat != nil && lon != nil { // a city CarTrak already looked up; nothing is fetched while a report is made
				if c, ok := s.S.GetPlace(geocode.Key(*lat, *lon)); ok {
					return c
				}
			}
			return ""
		}
		method := "Car"
		if pts, err := s.S.PointsBetween(t.StartTS, *t.EndTS); err == nil && len(pts) > 0 {
			method = "Phone GPS"
		}
		out = append(out, report.Trip{ID: t.ID, Start: time.Unix(t.StartTS, 0).In(l), End: time.Unix(*t.EndTS, 0).In(l), Zone: l.String(),
			Category: t.Category, DistanceKm: t.DistanceKm, Notes: t.Notes, From: name(o.StartPlace, t.StartLat, t.StartLon), To: name(o.EndPlace, t.EndLat, t.EndLon),
			Method: method, EnergyKwh: o.EnergyKwh, SoCStart: t.StartSoC, SoCEnd: t.EndSoC, OdoStartKm: t.StartOdo, OdoEndKm: t.EndOdo, TopSpeedKph: t.MaxSpeedKph})
	}
	return out
}

// downloadReport: GET /api/reports/trips.xlsx or /api/reports/trips.csv. The trips of the chosen categories and days, laid out as
// the standard report, as the columns asked for (?columns=value,distance,...), or as a saved custom report (?template=3).
//
//	categories=work,medical   which categories; empty or "all" means every one
//	from=2026-01-01&to=2026-10-20  or  weeks=26-28&year=2026   which days or calendar weeks
//	units=km|mi
func (s *Server) downloadReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	loc := s.defaultLoc()
	start, end, err := reportPeriod(q.Get("weeks"), q.Get("year"), q.Get("from"), q.Get("to"), loc)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	cats, _ := s.S.ListCategories()
	labels, rates := map[string]string{}, map[string]float64{}
	for _, c := range cats {
		labels[c.Key] = c.Label
		switch {
		case c.Rate != nil:
			rates[c.Key] = *c.Rate
		case c.Key == "work" && s.mileageRate() != nil:
			rates[c.Key] = *s.mileageRate()
		}
	}
	want := map[string]bool{}
	var names []string
	for _, k := range strings.Split(q.Get("categories"), ",") {
		if k = strings.TrimSpace(k); k != "" && k != "all" && labels[k] != "" {
			want[k] = true
			names = append(names, labels[k])
		}
	}
	purpose := "All"
	if len(names) > 0 {
		purpose = strings.Join(names, ", ")
	}
	opt := report.Options{Metric: q.Get("units") == "km", Labels: labels, Rates: rates, Summary: true, Person: userFrom(r).Name, Purpose: purpose,
		Range: start.Format("January 2, 2006") + " - " + end.AddDate(0, 0, -1).Format("January 2, 2006"), Vehicle: s.carName()}
	if t := q.Get("template"); t != "" {
		id, _ := strconv.ParseInt(t, 10, 64)
		tpl, _ := s.S.GetReportTemplate(id)
		if tpl == nil {
			fail(w, 404, "no such report")
			return
		}
		var c templateConfig
		if json.Unmarshal([]byte(tpl.Config), &c) != nil {
			fail(w, 500, "that report is damaged")
			return
		}
		opt.Columns, opt.Fixed, opt.Summary, opt.Title = c.Columns, c.Fixed, c.Summary, c.Title
		if opt.Title == "" {
			opt.Title = tpl.Name
		}
	} else if cols := strings.TrimSpace(q.Get("columns")); cols != "" {
		for _, k := range strings.Split(cols, ",") {
			if k = strings.TrimSpace(k); report.ValidKey(k) {
				opt.Columns = append(opt.Columns, report.Pick{Key: k})
			}
		}
	}
	opt.EffUnit = q.Get("eff")
	if c := s.capacityKwh(); c != nil {
		opt.Capacity = *c
	}
	opt.KwhPrice = s.periodKwhPrice(start.Unix(), end.Unix())
	all, err := s.S.TripsBetween(start.Unix(), end.Unix())
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	var picked []*store.Trip
	for _, t := range all {
		if len(want) == 0 || want[t.Category] {
			picked = append(picked, t)
		}
	}
	rows := s.reportTrips(picked, loc)
	name := fmt.Sprintf("cartrak-trips-%s-%s", start.Format("2006-01-02"), end.AddDate(0, 0, -1).Format("2006-01-02"))
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasSuffix(r.URL.Path, ".csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.csv"`)
		_ = report.CSV(w, rows, opt)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.xlsx"`)
	_ = report.Workbook(rows, opt).Write(w)
}

// carName is this car's name in the list of cars, or the kind of vehicle when there is a single car.
func (s *Server) carName() string {
	if s.CarID != 0 {
		if c, _ := s.core().GetCar(s.CarID); c != nil {
			return c.Name
		}
	}
	return s.vehicleInfo().Label
}

// periodKwhPrice is what a kWh that reached the battery cost over the period: the cost of the charging sessions in it divided by
// the energy they added, so it follows what you really paid. With no priced session it is the home price, else 0.
func (s *Server) periodKwhPrice(from, to int64) float64 {
	var cost, kwh float64
	if ss, err := s.chargeSessions(100000); err == nil {
		for _, c := range ss {
			if c.StartTS < from || c.StartTS >= to || c.Cost == nil || c.AddedKwh == nil || *c.AddedKwh <= 0 {
				continue
			}
			cost += *c.Cost
			kwh += *c.AddedKwh
		}
	}
	if kwh > 0 {
		return cost / kwh
	}
	if p := s.prices().Home; p != nil {
		return *p
	}
	return 0
}
