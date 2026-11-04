package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/afdc"
	"github.com/ZachCurry13/cartrak/internal/superchargers"
)

const (
	superKey    = "superchargers"
	superAtKey  = "superchargers_at"
	eaKey       = "afdc_ea"
	eaAtKey     = "afdc_ea_at"
	superMaxAge = 7 * 24 * time.Hour
)

var superMu sync.Mutex

// keptList is a charger list kept in the database and downloaded again when it is over a week old (or missing). If the download
// fails the old copy is used.
func (s *Server) keptList(key, atKey string, get func(context.Context) ([]superchargers.Site, error), ctx context.Context) ([]superchargers.Site, time.Time, error) {
	superMu.Lock()
	defer superMu.Unlock()
	var sites []superchargers.Site
	var at time.Time
	if b := s.S.GetKV(key); b != "" && json.Unmarshal([]byte(b), &sites) == nil {
		if n, _ := strconv.ParseInt(s.S.GetKV(atKey), 10, 64); n > 0 {
			at = time.Unix(n, 0)
		}
	}
	if len(sites) > 0 && time.Since(at) < superMaxAge {
		return sites, at, nil
	}
	fctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	fresh, err := get(fctx)
	if err != nil || len(fresh) == 0 {
		if len(sites) > 0 {
			return sites, at, nil
		}
		if err == nil {
			err = errors.New("the list was empty")
		}
		return nil, at, err
	}
	if b, merr := json.Marshal(fresh); merr == nil {
		_ = s.S.SetKV(key, string(b))
		_ = s.S.SetKV(atKey, strconv.FormatInt(time.Now().Unix(), 10))
	}
	return fresh, time.Now(), nil
}

func (s *Server) superSites(ctx context.Context) ([]superchargers.Site, time.Time, error) {
	url := s.SuperURL
	if url == "" {
		url = superchargers.DefaultURL
	}
	return s.keptList(superKey, superAtKey, func(c context.Context) ([]superchargers.Site, error) {
		return superchargers.Fetch(c, nil, url, "CarTrak/"+s.Version+" (+https://github.com/ZachCurry13/cartrak)")
	}, ctx)
}

func (s *Server) eaSites(ctx context.Context, key string) ([]superchargers.Site, error) {
	base := s.AFDCURL
	if base == "" {
		base = afdc.DefaultURL
	}
	sites, _, err := s.keptList(eaKey, eaAtKey, func(c context.Context) ([]superchargers.Site, error) { return afdc.Fetch(c, nil, base, key) }, ctx)
	return sites, err
}

// sourceCounts is how many sites each choice on the Long trip tab would offer, from the lists already kept (nothing is downloaded
// here), counted for North America where Superchargers use the NACS plug. ready is false until the lists have been fetched once.
func (s *Server) sourceCounts() map[string]any {
	inNA := func(lat, lon float64) bool { return lat > 14 && lat < 72 && lon > -170 && lon < -50 }
	var tesla, ea []superchargers.Site
	if b := s.S.GetKV(superKey); b != "" {
		_ = json.Unmarshal([]byte(b), &tesla)
	}
	if b := s.S.GetKV(eaKey); b != "" {
		_ = json.Unmarshal([]byte(b), &ea)
	}
	adapter, magic := 0, 0
	for _, x := range tesla {
		if !inNA(x.Lat, x.Lon) {
			continue
		}
		if x.NACS {
			adapter++
		}
		if x.Dock {
			magic++
		}
	}
	eaN := 0
	for _, x := range ea {
		if inNA(x.Lat, x.Lon) {
			eaN++
		}
	}
	return map[string]any{"ready": len(tesla) > 0, "adapter": adapter, "magic": magic, "ea": eaN, "eaReady": len(ea) > 0}
}

// tripSources: GET /api/trip-chain/sources. Admin only.
func (s *Server) tripSources(w http.ResponseWriter, r *http.Request) { ok(w, s.sourceCounts()) }
