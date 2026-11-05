package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

type tripShareOut struct {
	Expires   int64  `json:"expires"`
	Opens     int    `json:"opens"`
	HideStart bool   `json:"hideStart"`
	URL       string `json:"url"`
}

type savedTripOut struct {
	store.SavedTrip
	Share *tripShareOut `json:"share"`
}

// shareTrip: POST /api/saved-trips/{id}/share {"days": 7, "hideStart": true}. Makes a link that shows a saved trip as a plain page, with
// no sign-in. By default it leaves out where the trip starts, which is usually home. A new link replaces the old one. Admin only.
func (s *Server) shareTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var in struct {
		Days      int   `json:"days"`
		HideStart *bool `json:"hideStart"`
	}
	_ = decode(r, &in)
	if in.Days < 1 || in.Days > 90 {
		in.Days = 7
	}
	hide := in.HideStart == nil || *in.HideStart
	trips, _ := s.S.SavedTrips()
	found := false
	for _, t := range trips {
		found = found || t.ID == id
	}
	if !found {
		fail(w, 404, "no such trip")
		return
	}
	tok := shortToken(20)
	if err := s.S.SetTripShare(id, tok, store.TokenHash(tok), time.Now().AddDate(0, 0, in.Days).Unix(), hide); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]any{"url": s.publicBase() + "/t/" + tok, "path": "/t/" + tok, "addressKnown": s.publicBase() != ""})
}

// unshareTrip: DELETE /api/saved-trips/{id}/share.
func (s *Server) unshareTrip(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.ClearTripShare(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, map[string]bool{"ok": true})
}

type planStop struct{ Time, Name, Chip, Detail, Kind string }
type planDay struct {
	Name, Date, Summary, Maps string
	Stops                     []planStop
}

var planPage = template.Must(template.New("plan").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow"><title>{{.Title}}</title><style>
:root{--bg:#f3f5f8;--card:#fff;--text:#14191f;--muted:#5b6672;--line:#dde3ea;--accent:#0a4bb0;--on:#fff;--ok:#07694e;--work:#9a4308;color-scheme:light}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--card:#161d27;--text:#e6edf3;--muted:#8b98a5;--line:#232d3a;--accent:#6aa5ff;--on:#05122b;--ok:#2fd4a0;--work:#f5b84a;color-scheme:dark}}
body{margin:0;background:var(--bg);color:var(--text);font:16px/1.45 system-ui,sans-serif}main{max-width:640px;margin:0 auto;padding:16px}
h1{font-size:1.5rem;margin:8px 0}h2{font-size:1.1rem;margin:0 0 4px}section{background:var(--card);border-radius:20px;padding:16px;margin:12px 0}
.muted{color:var(--muted)}ol{padding-left:0;list-style:none;margin:8px 0}li{padding:10px 0;border-top:1px solid var(--line)}li:first-child{border-top:0}
.chip{font-size:.8rem;border:1px solid currentColor;border-radius:999px;padding:1px 8px;margin-left:6px}.charge .chip{color:var(--ok)}.overnight .chip{color:var(--work)}
a.btn{display:inline-block;background:var(--accent);color:var(--on);text-decoration:none;border-radius:999px;padding:12px 20px;min-height:24px;font-weight:600}
</style></head><body><main><h1>{{.Title}}</h1><p class="muted">{{.Sub}}</p>
{{range .Days}}<section><h2>{{.Name}} <span class="muted">{{.Date}}</span></h2><p class="muted">{{.Summary}}</p><ol>{{range .Stops}}<li class="{{.Kind}}"><b>{{.Time}}</b> {{.Name}}{{if .Chip}}<span class="chip">{{.Chip}}</span>{{end}}{{if .Detail}}<div class="muted">{{.Detail}}</div>{{end}}</li>{{end}}</ol>
<p><a class="btn" href="{{.Maps}}" rel="noopener">Open in Google Maps</a></p></section>{{end}}
<p class="muted">Planned with CarTrak. Times are estimates; a charger can be busy or out of service, so check it before you go.</p></main></body></html>`))

// mapsParts is a day's points as Google Maps directions links, ten points to a link (it takes that many at a time).
func mapsParts(pts [][2]float64) []string {
	var out []string
	for i := 0; i+1 < len(pts); i += 9 {
		end := min(i+10, len(pts))
		part := pts[i:end]
		q := url.Values{"api": {"1"}, "travelmode": {"driving"}, "destination": {fmt.Sprintf("%.6f,%.6f", part[len(part)-1][0], part[len(part)-1][1])}}
		if len(part) > 2 {
			var w []string
			for _, p := range part[1 : len(part)-1] {
				w = append(w, fmt.Sprintf("%.6f,%.6f", p[0], p[1]))
			}
			q.Set("waypoints", strings.Join(w, "|"))
		}
		link := "https://www.google.com/maps/dir/?" + q.Encode()
		if !(i == 0 && pts[0] == [2]float64{}) { // an empty first point is a hidden start: leave the origin to the viewer's own position
			link += fmt.Sprintf("&origin=%.6f,%.6f", part[0][0], part[0][1])
		}
		out = append(out, link)
	}
	return out
}

// tripPage: GET /t/{token}. A saved trip as a plain page. It has no scripts and loads nothing from anywhere.
func (s *Server) tripPage(w http.ResponseWriter, r *http.Request) {
	ip := s.clientIP(r)
	shareHeaders(w)
	if s.lim.blocked(ip) {
		http.Error(w, "too many attempts, wait a minute", http.StatusTooManyRequests)
		return
	}
	t, hide, err := s.S.TripByShare(store.TokenHash(r.PathValue("token")), time.Now().Unix())
	if err != nil || t == nil {
		s.lim.fail(ip)
		http.NotFound(w, r)
		return
	}
	var saved struct {
		Body struct {
			Dest struct{ Name string } `json:"dest"`
		} `json:"body"`
		Result struct {
			Days     []tripchain.Day `json:"days"`
			Currency string          `json:"currency"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(t.Body), &saved) != nil || len(saved.Result.Days) == 0 {
		http.NotFound(w, r)
		return
	}
	loc, metric := s.defaultLoc(), s.Cfg != nil && s.Cfg.Value(settings.FuelUnitsKey) == "metric"
	distance := func(km float64) string {
		if metric {
			return fmt.Sprintf("%.0f km", km)
		}
		return fmt.Sprintf("%.0f mi", km*0.621371)
	}
	span := func(min float64) string { m := int(min + 0.5); return fmt.Sprintf("%dh %02dm", m/60, m%60) }
	chip := map[string]string{"charge": "Charge", "overnight": "Overnight", "dest": "Arrive", "via": "Stop"}
	out := struct {
		Title, Sub string
		Days       []planDay
	}{Title: t.Name, Sub: fmt.Sprintf("%d day%s to %s", len(saved.Result.Days), map[bool]string{true: "", false: "s"}[len(saved.Result.Days) == 1], saved.Body.Dest.Name)}
	for i, d := range saved.Result.Days {
		day := planDay{Name: map[bool]string{true: "The trip", false: fmt.Sprintf("Day %d", i+1)}[len(saved.Result.Days) == 1],
			Date:    time.Unix(d.DepartTS, 0).In(loc).Format("Mon, Jan 2"),
			Summary: fmt.Sprintf("%s · %s driving + about %s charging · %d charge stop%s", distance(d.Km), span(d.DriveMin), span(d.ChargeMin), d.ChargeStops, map[bool]string{true: "", false: "s"}[d.ChargeStops == 1])}
		var pts [][2]float64
		for j, st := range d.Stops {
			hidden := hide && i == 0 && j == 0
			at := time.Unix(d.DepartTS+int64(st.ArriveMin*60), 0).In(loc).Format("3:04 PM")
			ps := planStop{Time: at, Name: st.Name, Chip: chip[st.Kind], Kind: st.Kind}
			if st.Kind == "start" || j == 0 {
				ps.Time, ps.Chip = time.Unix(d.DepartTS, 0).In(loc).Format("3:04 PM"), ""
				ps.Name = "Leave" + map[bool]string{true: "", false: ": " + st.Name}[hidden || st.Name == ""]
				if hidden {
					ps.Name = "Leave"
				}
			}
			switch {
			case st.ChargeMin > 0:
				ps.Detail = fmt.Sprintf("%.0f kWh · %.0f min · battery %.0f%% to %.0f%%", st.Kwh, st.ChargeMin, st.ArrivePct, st.DepartPct)
				if st.Cost != nil && saved.Result.Currency != "" {
					ps.Detail += fmt.Sprintf(" · about %s%.2f", saved.Result.Currency, *st.Cost)
				}
			case j > 0:
				ps.Detail = fmt.Sprintf("%s from the last stop · battery %.0f%%", distance(st.LegKm), st.ArrivePct)
			}
			if hidden {
				pts = append(pts, [2]float64{})
			} else {
				pts = append(pts, [2]float64{st.Lat, st.Lon})
			}
			day.Stops = append(day.Stops, ps)
		}
		if parts := mapsParts(pts); len(parts) > 0 {
			day.Maps = parts[0]
		}
		out.Days = append(out.Days, day)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")
	_ = planPage.Execute(w, out)
}
