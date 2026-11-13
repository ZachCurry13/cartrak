package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/calendar"
	"github.com/ZachCurry13/cartrak/internal/companylist"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// calCache keeps the calendar for calKeep, and the last good copy when a fetch fails.
type calCache struct {
	mu     sync.Mutex
	at     time.Time
	addr   string
	events []calendar.Event
	err    string
}

const calKeep = 30 * time.Minute

// calendarEvents is the calendar's events, fetched again when older than calKeep. ok is false when no calendar is set.
func (s *Server) calendarEvents(ctx context.Context) (ev []calendar.Event, errText string, ok bool) {
	if s.Cfg == nil || s.Cfg.CalendarURL() == "" {
		return nil, "", false
	}
	addr := s.Cfg.CalendarURL()
	c := &s.cal
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.addr == addr && time.Since(c.at) < calKeep {
		return c.events, c.err, true
	}
	text, err := calendar.Fetch(ctx, nil, addr)
	c.at = time.Now()
	if err != nil {
		if c.addr != addr {
			c.events = nil
		}
		c.addr, c.err = addr, err.Error()
		return c.events, c.err, true
	}
	c.addr, c.err, c.events = addr, "", calendar.Parse(text, s.Cfg.Location())
	return c.events, "", true
}

// appointment is one appointment of a day, with the place CarTrak found for it.
type appointment struct {
	Title    string       `json:"title"`
	Location string       `json:"location"`
	Start    int64        `json:"start"`
	End      int64        `json:"end"`
	AllDay   bool         `json:"allDay"`
	Place    *companyPick `json:"place"`
	From     string       `json:"from,omitempty"` // account, place or search: how the place was found
}

// appointments lists a day's appointments (midnight to midnight in the car's zone) and finds their places: one of the remembered
// accounts or saved places by its title or location, else the location searched like an address (each location searched once).
func (s *Server) appointments(ctx context.Context, day time.Time) ([]appointment, string, bool) {
	ev, errText, ok := s.calendarEvents(ctx)
	if !ok {
		return nil, "", false
	}
	loc := s.Cfg.Location()
	from := time.Date(day.In(loc).Year(), day.In(loc).Month(), day.In(loc).Day(), 0, 0, 0, 0, loc)
	saved, _ := s.S.ListNamedPlaces()
	known := s.accounts()
	out := []appointment{}
	for _, o := range calendar.Between(ev, from, from.AddDate(0, 0, 1)) {
		a := appointment{Title: o.Title, Location: o.Location, Start: o.Start.Unix(), End: o.End.Unix(), AllDay: o.AllDay}
		if !o.AllDay {
			a.Place, a.From = s.placeFor(ctx, o, known, saved)
		}
		out = append(out, a)
	}
	return out, errText, true
}

func (s *Server) placeFor(ctx context.Context, o calendar.Occurrence, known []account, saved []*store.NamedPlace) (*companyPick, string) {
	for _, name := range []string{o.Location, o.Title} {
		if name == "" {
			continue
		}
		if a := byName(known, name); a != nil {
			p := a.companyPick
			return &p, "account"
		}
		for _, p := range saved {
			if companylist.Same(name, p.Name) || strings.EqualFold(strings.TrimSpace(name), p.Name) {
				return &companyPick{Name: p.Name, Lat: p.Lat, Lon: p.Lon}, "place"
			}
		}
	}
	if strings.TrimSpace(o.Location) == "" {
		return nil, ""
	}
	key := "calgeo:" + strings.ToLower(strings.TrimSpace(o.Location))
	var hit struct {
		companyPick
		None int64 `json:"none"`
	}
	if json.Unmarshal([]byte(s.S.GetKV(key)), &hit) == nil {
		if hit.Name != "" {
			p := hit.companyPick
			return &p, "search"
		}
		if time.Since(time.Unix(hit.None, 0)) < 24*time.Hour { // not found yesterday: not asked again until tomorrow
			return nil, ""
		}
	}
	c := s.geoClient()
	if c == nil {
		return nil, ""
	}
	found, err := c.SearchIn(ctx, companylist.SearchName(o.Location), s.homeNear(), s.lookupCountry())
	if err != nil {
		return nil, "" // asked again next time
	}
	if found = withinReach(found); len(found) == 0 {
		b, _ := json.Marshal(map[string]int64{"none": time.Now().Unix()})
		_ = s.S.SetKV(key, string(b))
		return nil, ""
	}
	p := companyPick{Name: firstNonEmpty(o.Title, found[0].Name), Address: found[0].Address, Lat: found[0].Lat, Lon: found[0].Lon}
	b, _ := json.Marshal(p)
	_ = s.S.SetKV(key, string(b))
	return &p, "search"
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// homeNear is Home's position, or the car's, to search near.
func (s *Server) homeNear() *[2]float64 {
	saved, _ := s.S.ListNamedPlaces()
	for _, p := range saved {
		if strings.EqualFold(p.Name, "home") {
			return &[2]float64{p.Lat, p.Lon}
		}
	}
	if x, _ := s.S.LatestSample(); x != nil && x.Lat != nil && x.Lon != nil {
		return &[2]float64{*x.Lat, *x.Lon}
	}
	return nil
}

func (s *Server) lookupCountry() string {
	if s.Cfg != nil && s.Cfg.LiveGasUnits() {
		return "us"
	}
	return ""
}

// calendarDay: GET /api/calendar/day?date=2026-11-16 (today when left out). Admin only: appointments are private.
func (s *Server) calendarDay(w http.ResponseWriter, r *http.Request) {
	day := time.Now()
	if d := r.URL.Query().Get("date"); d != "" {
		t, err := time.ParseInLocation("2006-01-02", d, s.Cfg.Location())
		if err != nil {
			fail(w, 400, "the date is written 2026-11-16")
			return
		}
		day = t
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	list, errText, on := s.appointments(ctx, day)
	ok(w, map[string]any{"connected": on, "error": errText, "events": list, "date": day.In(s.Cfg.Location()).Format("2006-01-02")})
}
