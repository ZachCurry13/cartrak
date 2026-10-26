package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	cols := report.Columns
	if s.fuelCar() { // no kWh, battery or charging-cost columns to offer a gas or hybrid car
		cols = nil
		for _, c := range report.Columns {
			switch c.Key {
			case "energy", "efficiency", "energycost", "costper", "mpge", "socstart", "socend":
			default:
				cols = append(cols, c)
			}
		}
	}
	ok(w, map[string]any{"columns": cols, "categories": cats, "templates": out})
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
