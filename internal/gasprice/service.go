package gasprice

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

// KV is where fetched prices are remembered between runs.
type KV interface {
	GetKV(k string) string
	SetKV(k, v string) error
}

const (
	refreshAfter = 12 * time.Hour
	historyFrom  = "2022-01-01" // the oldest week asked for, so earlier charges can be priced too
)

// Price is the average price that applied to one place and week.
type Price struct {
	PerGallon float64
	Area      string // "Minnesota average"
	Week      string // the Monday it is the average for
}

type blob struct {
	Fetched int64   `json:"t"`
	Points  []Point `json:"p"`
}

// Service hands out prices from what it has remembered and refreshes them in the background, so a page never waits on the EIA.
type Service struct {
	KV      KV
	HTTP    *http.Client // optional
	BaseURL string       // empty means DefaultURL
	Now     func() time.Time

	mu    sync.Mutex
	busy  map[string]bool
	known map[string]bool // areas in use, kept so they can be refreshed on a schedule
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func cacheKey(a Area) string { return "gasprice:" + a.ID }

func (s *Service) load(a Area) blob {
	var b blob
	_ = json.Unmarshal([]byte(s.KV.GetKV(cacheKey(a))), &b)
	return b
}

// Refresh fetches an area's weekly history and remembers it.
func (s *Service) Refresh(ctx context.Context, key string, a Area) error {
	since, _ := time.Parse("2006-01-02", historyFrom)
	pts, err := Fetch(ctx, s.HTTP, s.BaseURL, key, a, since)
	s.note(err)
	if err != nil {
		return err
	}
	if len(pts) == 0 {
		return nil
	}
	raw, _ := json.Marshal(blob{Fetched: s.now().Unix(), Points: pts})
	return s.KV.SetKV(cacheKey(a), string(raw))
}

// At is the price that applied in an area on a day, from what is remembered. It starts a background refresh when the
// remembered prices are missing or old, and reports false until there is something to say.
func (s *Service) At(key string, a Area, ts int64) (Price, bool) {
	if key == "" {
		return Price{}, false
	}
	s.remember(a)
	b := s.load(a)
	if s.now().Sub(time.Unix(b.Fetched, 0)) > refreshAfter {
		s.refreshLater(key, a)
	}
	day := time.Unix(ts, 0).UTC().Format("2006-01-02")
	i := sort.Search(len(b.Points), func(i int) bool { return b.Points[i].Date > day }) - 1
	if i < 0 {
		return Price{}, false
	}
	return Price{PerGallon: b.Points[i].Price, Area: a.Label, Week: b.Points[i].Date}, true
}

func (s *Service) refreshLater(key string, a Area) {
	s.mu.Lock()
	if s.busy == nil {
		s.busy = map[string]bool{}
	}
	if s.busy[a.ID] {
		s.mu.Unlock()
		return
	}
	s.busy[a.ID] = true
	s.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = s.Refresh(ctx, key, a)
		s.mu.Lock()
		delete(s.busy, a.ID)
		s.mu.Unlock()
	}()
}
