package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/afdc"
	"github.com/ZachCurry13/cartrak/internal/superchargers"
)

const (
	dataStateKey   = "dataupdates"
	dataCheckEvery = 24 * time.Hour
	validatorKey   = "super_validator"
)

// dataset is a list CarTrak downloads and keeps, so it can say when a newer one is out and fetch it when asked. Adding a downloaded
// catalog later means adding one entry to datasets().
type dataset struct {
	ID, Name, Source string
	count            func() int
	at               func() time.Time
	check            func(ctx context.Context) (available bool, note string)
	update           func(ctx context.Context) error
}

type dataState struct {
	CheckedAt int64             `json:"checkedAt"`
	Avail     map[string]string `json:"avail"` // the datasets with something newer, and why
}

var dataBusy sync.Mutex

func (s *Server) loadDataState() dataState {
	var st dataState
	_ = json.Unmarshal([]byte(s.S.GetKV(dataStateKey)), &st)
	if st.Avail == nil {
		st.Avail = map[string]string{}
	}
	return st
}

func (s *Server) saveDataState(st dataState) {
	b, _ := json.Marshal(st)
	_ = s.S.SetKV(dataStateKey, string(b))
}

// superValidator is what the list's server says identifies the current list (its ETag, last change or size), or "".
func (s *Server) superValidator(ctx context.Context) string {
	url := s.SuperURL
	if url == "" {
		url = superchargers.DefaultURL
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "CarTrak/"+s.Version+" (+https://github.com/ZachCurry13/cartrak)")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return ""
	}
	resp.Body.Close()
	if v := resp.Header.Get("ETag"); v != "" {
		return v
	}
	if v := resp.Header.Get("Last-Modified"); v != "" {
		return v
	}
	return resp.Header.Get("Content-Length")
}

func (s *Server) datasets() []dataset {
	listAge := func(key, atKey string) (func() int, func() time.Time) {
		return func() int { n, _, _ := s.knownList(key, atKey); return n }, func() time.Time { _, t, _ := s.knownList(key, atKey); return t }
	}
	sCount, sAt := listAge(superKey, superAtKey)
	eCount, eAt := listAge(eaKey, eaAtKey)
	old := func(t time.Time, days int) bool {
		return t.IsZero() || time.Since(t) > time.Duration(days)*24*time.Hour
	}
	return []dataset{
		{ID: "super", Name: "Supercharger list", Source: "supercharge.info", count: sCount, at: sAt,
			check: func(ctx context.Context) (bool, string) {
				if sCount() == 0 {
					return false, "Not downloaded yet: it is fetched the first time you plan a long trip."
				}
				v, kept := s.superValidator(ctx), s.S.GetKV(validatorKey)
				switch {
				case v != "" && kept != "" && v != kept:
					return true, "supercharge.info has a newer list."
				case v == "" && old(sAt(), 7):
					return true, "It is over a week old."
				}
				return false, ""
			},
			update: func(ctx context.Context) error {
				url := s.SuperURL
				if url == "" {
					url = superchargers.DefaultURL
				}
				sites, err := superchargers.Fetch(ctx, nil, url, "CarTrak/"+s.Version+" (+https://github.com/ZachCurry13/cartrak)")
				if err != nil {
					return err
				}
				b, _ := json.Marshal(sites)
				_ = s.S.SetKV(superKey, string(b))
				_ = s.S.SetKV(superAtKey, strconv.FormatInt(time.Now().Unix(), 10))
				_ = s.S.SetKV(validatorKey, s.superValidator(ctx))
				return nil
			}},
		{ID: "ea", Name: "Electrify America list", Source: "NLR charger list", count: eCount, at: eAt,
			check: func(context.Context) (bool, string) {
				if s.Cfg == nil || s.Cfg.AFDCKey() == "" || eCount() == 0 {
					return false, ""
				}
				return old(eAt(), 30), "It is over a month old."
			},
			update: func(ctx context.Context) error {
				if s.Cfg == nil || s.Cfg.AFDCKey() == "" {
					return errNeedKey
				}
				base := s.AFDCURL
				if base == "" {
					base = afdc.DefaultURL
				}
				sites, err := afdc.Fetch(ctx, nil, base, s.Cfg.AFDCKey())
				if err != nil {
					return err
				}
				b, _ := json.Marshal(sites)
				_ = s.S.SetKV(eaKey, string(b))
				_ = s.S.SetKV(eaAtKey, strconv.FormatInt(time.Now().Unix(), 10))
				return nil
			}},
		{ID: "gas", Name: "Gas prices", Source: "EIA", count: func() int { return 0 }, at: func() time.Time { return time.Unix(s.gasLiveStatus().OK, 0) },
			check: func(context.Context) (bool, string) {
				if s.Cfg == nil || s.Cfg.EIAKey() == "" {
					return false, ""
				}
				return old(time.Unix(s.gasLiveStatus().OK, 0), 8), "The weekly prices have not been fetched for over a week."
			},
			update: func(ctx context.Context) error {
				if s.Cfg == nil || s.Cfg.EIAKey() == "" {
					return errNeedKey
				}
				s.gasService().RefreshKnown(ctx, s.Cfg.EIAKey())
				if st := s.gasLiveStatus(); st.Err != "" {
					return errString(st.Err)
				}
				return nil
			}},
	}
}

type errString string

func (e errString) Error() string { return string(e) }

const errNeedKey = errString("add the key first (More, Integrations)")

// checkDatasets asks each list whether something newer is out and keeps the answer.
func (s *Server) checkDatasets(ctx context.Context) dataState {
	st := dataState{CheckedAt: time.Now().Unix(), Avail: map[string]string{}}
	for _, d := range s.datasets() {
		if avail, note := d.check(ctx); avail {
			st.Avail[d.ID] = note
		}
	}
	s.saveDataState(st)
	return st
}

// dueCheck starts the daily check in the background when it is due, so opening the app is what triggers it and nothing waits.
func (s *Server) dueCheck() {
	if time.Since(time.Unix(s.loadDataState().CheckedAt, 0)) < dataCheckEvery || !dataBusy.TryLock() {
		return
	}
	go func() {
		defer dataBusy.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		s.checkDatasets(ctx)
	}()
}

// dataUpdatesAvailable counts the lists with something newer, and a newer CarTrak, for the dot on More.
func (s *Server) dataUpdatesAvailable() int {
	s.dueCheck()
	n := len(s.loadDataState().Avail)
	if s.Upd != nil && s.core().GetKV("update_check") != "off" && s.Upd.Info().Available {
		n++
	}
	return n
}

// dataUpdatesView: GET /api/data-updates. Admin only.
func (s *Server) dataUpdatesView(w http.ResponseWriter, r *http.Request) {
	s.dueCheck()
	st := s.loadDataState()
	items := []map[string]any{}
	for _, d := range s.datasets() {
		var at int64
		if t := d.at(); !t.IsZero() && t.Unix() > 0 {
			at = t.Unix()
		}
		note, avail := st.Avail[d.ID]
		items = append(items, map[string]any{"id": d.ID, "name": d.Name, "source": d.Source, "count": d.count(), "updatedAt": at, "available": avail, "note": note})
	}
	app := map[string]any{"current": s.Version, "enabled": false}
	if s.Upd != nil {
		info := s.Upd.Info()
		app = map[string]any{"current": s.Version, "enabled": s.core().GetKV("update_check") != "off", "available": info.Available, "latest": info.Latest, "url": info.URL}
	}
	ok(w, map[string]any{"items": items, "app": app, "checkedAt": st.CheckedAt})
}

// dataUpdatesCheck: POST /api/data-updates/check. Looks now instead of waiting for the daily check.
func (s *Server) dataUpdatesCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	if s.Upd != nil && s.core().GetKV("update_check") != "off" {
		_ = s.Upd.Check(ctx)
	}
	s.checkDatasets(ctx)
	s.dataUpdatesView(w, r)
}

// dataUpdatesRun: POST /api/data-updates/{id}/update. Fetches that list now.
func (s *Server) dataUpdatesRun(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	for _, d := range s.datasets() {
		if d.ID != r.PathValue("id") {
			continue
		}
		if err := d.update(ctx); err != nil {
			fail(w, upstreamFailed, d.Name+" could not be updated: "+err.Error())
			return
		}
		st := s.loadDataState()
		delete(st.Avail, d.ID)
		s.saveDataState(st)
		s.dataUpdatesView(w, r)
		return
	}
	fail(w, 404, "no such list")
}
