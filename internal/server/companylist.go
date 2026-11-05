package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/companylist"
	"github.com/ZachCurry13/cartrak/internal/geocode"
)

type companyPick struct {
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

type companyRow struct {
	companylist.Entry
	Status  string          `json:"status"` // remembered, saved, found (the search agrees), check (pick one) or none
	Pick    *companyPick    `json:"pick"`
	Options []geocode.Match `json:"options"`
	Error   string          `json:"error,omitempty"` // the lookup service did not answer: not the same as "nothing found"
}

// resolveCompanies reads a pasted list and finds each company: POST /api/company-list/resolve {"text": "..."}. A company is
// found first by the number on its line (one confirmed before), then among the saved places by name, then by address search
// near Home. Only the searched names leave the server, to the same lookup service as the address search. Admin only.
func (s *Server) resolveCompanies(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if decode(r, &in) != nil || len(in.Text) > 10000 {
		fail(w, 400, "paste the list as text")
		return
	}
	entries, more := companylist.Parse(in.Text)
	if len(entries) == 0 {
		fail(w, 400, "No companies found in that text")
		return
	}
	saved, _ := s.S.ListNamedPlaces()
	var near *[2]float64
	for _, p := range saved {
		if strings.EqualFold(p.Name, "home") {
			near = &[2]float64{p.Lat, p.Lon}
		}
	}
	if x, _ := s.S.LatestSample(); near == nil && x != nil && x.Lat != nil && x.Lon != nil {
		near = &[2]float64{*x.Lat, *x.Lon}
	}
	c := s.geoClient()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	rows := make([]companyRow, 0, len(entries))
	for _, e := range entries {
		row := companyRow{Entry: e, Status: "none", Options: []geocode.Match{}}
		var mem companyPick
		if e.Ref != "" && json.Unmarshal([]byte(s.S.GetKV("company:"+e.Ref)), &mem) == nil && mem.Name != "" {
			row.Status, row.Pick = "remembered", &mem
		}
		for _, p := range saved {
			if row.Pick == nil && companylist.Same(e.Name, p.Name) {
				row.Status, row.Pick = "saved", &companyPick{Name: p.Name, Lat: p.Lat, Lon: p.Lon}
			}
		}
		if row.Pick == nil && c != nil {
			found, err := c.Search(ctx, companylist.SearchName(e.Name), near)
			if (err != nil || len(found) == 0) && companylist.SearchName(e.Name) != e.Name { // the name as written, in case cleaning it hid it
				found, err = c.Search(ctx, e.Name, near)
			}
			if err != nil {
				row.Error = err.Error()
			}
			if err == nil && len(found) > 0 {
				row.Options, row.Status = found, "check"
				if companylist.Same(e.Name, found[0].Name) {
					row.Status, row.Pick = "found", &companyPick{Name: e.Name, Address: found[0].Address, Lat: found[0].Lat, Lon: found[0].Lon}
				}
			}
		}
		rows = append(rows, row)
	}
	ok(w, map[string]any{"rows": rows, "more": more, "lookupOff": c == nil})
}

// rememberCompany keeps the place chosen for a company's number, so the next list finds it by itself:
// POST /api/company-list/remember. Admin only.
func (s *Server) rememberCompany(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Ref string `json:"ref"`
		companyPick
	}
	if decode(r, &in) != nil || in.Ref == "" || len(in.Ref) > 20 || strings.Trim(in.Ref, "0123456789") != "" || in.Name == "" || len(in.Name) > 200 || in.Lat < -90 || in.Lat > 90 || in.Lon < -180 || in.Lon > 180 {
		fail(w, 400, "give the number, the name and the position")
		return
	}
	b, _ := json.Marshal(in.companyPick)
	if err := s.S.SetKV("company:"+in.Ref, string(b)); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, map[string]bool{"ok": true})
}
