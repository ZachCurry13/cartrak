package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/journey"
	"github.com/ZachCurry13/cartrak/internal/share"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// shareRoutes returns the route of each leg as it may be shown: trimmed at both ends, or whole, thinned out.
// For stats-only links it returns nothing.
func (s *Server) shareRoutes(sh *store.Share, ts []*store.Trip) [][][2]float64 {
	if sh.Mode == "stats" {
		return nil
	}
	var out [][][2]float64
	for _, t := range ts {
		pts := s.routeOf(t)
		if sh.Mode == "trimmed" {
			pts = share.Trim(pts, share.TrimMeters)
		}
		if len(pts) >= 2 {
			out = append(out, share.Thin(pts, 250))
		}
	}
	return out
}

// validDay says whether a number is a date written YYYYMMDD, which is how a day is named in a link.
func validDay(id int64) bool {
	y, m, d := int(id/10000), int(id/100%100), int(id%100)
	if y < 2000 || y > 2100 || m < 1 || m > 12 || d < 1 || d > 31 {
		return false
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC).Day() == d
}

// shareTrips is the trips a link covers: one trip, all the legs of a journey, or every trip of one day.
func (s *Server) shareTrips(kind string, id int64) []*store.Trip {
	if kind == "day" {
		if !validDay(id) {
			return nil
		}
		start := time.Date(int(id/10000), time.Month(id/100%100), int(id%100), 0, 0, 0, 0, s.defaultLoc())
		ts, _ := s.S.TripsBetween(start.Unix(), start.AddDate(0, 0, 1).Unix())
		return ts
	}
	if kind == "journey" {
		ts, _ := s.S.JourneyTrips(id)
		return ts
	}
	if t, err := s.S.GetTrip(id); err == nil && t != nil {
		return []*store.Trip{t}
	}
	return nil
}

type shareIn struct {
	Kind        string `json:"kind"`
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Mode        string `json:"mode"`
	ShowCities  bool   `json:"showCities"`
	ShowTime    bool   `json:"showTime"`
	Units       string `json:"units"`
	ExpiresDays int    `json:"expiresDays"`
}

// createShare makes a public link and returns it once; only a hash of its secret is kept. Admin only.
func (s *Server) createShare(w http.ResponseWriter, r *http.Request) {
	var in shareIn
	if decode(r, &in) != nil || (in.Kind != "trip" && in.Kind != "journey" && in.Kind != "day") || (in.Mode != "stats" && in.Mode != "trimmed" && in.Mode != "full") || in.ExpiresDays < 0 || in.ExpiresDays > 365 {
		fail(w, 400, "choose what to share, how much of the route, and a valid expiry")
		return
	}
	if in.Units != "km" {
		in.Units = "mi"
	}
	ts := s.shareTrips(in.Kind, in.ID)
	if len(ts) == 0 {
		fail(w, 404, "nothing to share")
		return
	}
	for _, t := range ts {
		if t.Open || t.EndTS == nil {
			fail(w, http.StatusConflict, "a trip that is still in progress cannot be shared")
			return
		}
	}
	title := strings.TrimSpace(in.Title)
	if len(title) > 80 {
		title = title[:80]
	}
	sh := &store.Share{Kind: in.Kind, TargetID: in.ID, Title: title, Mode: in.Mode, ShowTime: in.ShowTime, ShowCities: in.ShowCities, Units: in.Units}
	if in.ExpiresDays > 0 {
		sh.Expires = time.Now().AddDate(0, 0, in.ExpiresDays).Unix()
	}
	sh.BBox = share.BBox(s.shareRoutes(sh, ts)...)
	if in.ShowCities {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		first, last := ts[0], ts[len(ts)-1]
		sh.Origin, sh.Dest = s.cityOf(ctx, first.StartLat, first.StartLon), s.cityOf(ctx, last.EndLat, last.EndLon)
	}
	tok := "sh_" + randToken()
	sh.TokenHash = store.TokenHash(tok)
	if err := s.S.CreateShare(sh); err != nil {
		fail(w, 500, "db error")
		return
	}
	base := ""
	if s.Cfg != nil {
		if a := s.Cfg.Value(keyAddress); a != "" {
			base = "https://" + a
		}
	}
	ok(w, map[string]any{"id": sh.ID, "path": "/s/" + tok, "url": base + "/s/" + tok, "publicAddressKnown": base != ""})
}

func (s *Server) cityOf(ctx context.Context, lat, lon *float64) string {
	if lat == nil || lon == nil {
		return ""
	}
	pl := s.places()
	if c, ok := pl.Cached(*lat, *lon); ok {
		return c
	}
	c, _ := pl.Lookup(ctx, *lat, *lon)
	return c
}

func (s *Server) listShares(w http.ResponseWriter, r *http.Request) {
	xs, err := s.S.ListShares()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, xs)
}

func (s *Server) revokeShare(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.S.RevokeShare(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "revoked")
}

// shareData is exactly what a public link shows. It never has the VIN, the car's model, live position, or
// anything the link was not set up to share.
func (s *Server) shareData(sh *store.Share) map[string]any {
	ts := s.shareTrips(sh.Kind, sh.TargetID)
	out := map[string]any{"title": sh.Title, "kind": sh.Kind, "mode": sh.Mode, "units": sh.Units, "legs": len(ts), "mapsOn": s.mapsOn() && sh.Mode != "stats"}
	if sh.ShowCities {
		out["origin"], out["dest"] = sh.Origin, sh.Dest
	}
	sum := journey.Summarize(ts, s.capacityKwh(), nil, nil)
	out["distanceKm"], out["durationSec"], out["energyKwh"], out["energyEstimated"], out["kmPerKwh"] = sum.DistanceKm, sum.MovingSec, sum.KwhUsed, sum.KwhEstimated, sum.KmPerKwh
	if sh.Kind == "journey" || sh.Kind == "day" {
		out["stops"], out["elapsedSec"] = len(sum.Stops), sum.ElapsedSec
	}
	if sh.ShowTime && len(ts) > 0 {
		out["start"], out["end"] = sum.StartTS, sum.EndTS
		out["tz"] = ts[0].TZ
	}
	routes := s.shareRoutes(sh, ts)
	out["routes"] = routes
	out["routeHidden"] = sh.Mode != "stats" && len(routes) == 0
	return out
}
