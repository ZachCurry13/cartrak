package gasprice

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	areasKey  = "gasprice:areas"
	statusKey = "gasprice:status"
)

// Status is how the last attempts to fetch prices went, for the Status screen.
type Status struct {
	OK    int64  `json:"ok"`    // when prices were last fetched successfully, Unix seconds; 0 when never
	Tried int64  `json:"tried"` // when the last attempt was made
	Err   string `json:"err"`   // why the last attempt failed; empty when it worked
}

// Status is how the last fetch went.
func (s *Service) Status() Status {
	var st Status
	_ = json.Unmarshal([]byte(s.KV.GetKV(statusKey)), &st)
	return st
}

func (s *Service) note(err error) {
	st := s.Status()
	st.Tried = s.now().Unix()
	if err == nil {
		st.OK, st.Err = st.Tried, ""
	} else {
		st.Err = err.Error()
	}
	raw, _ := json.Marshal(st)
	_ = s.KV.SetKV(statusKey, string(raw))
}

func areaByID(id string) (Area, bool) {
	for _, m := range []map[string]Area{stateAreas, regionAreas} {
		for _, a := range m {
			if a.ID == id {
				return a, true
			}
		}
	}
	return National, id == National.ID
}

// remember keeps the list of areas in use, so they can be refreshed without anyone asking for a price.
func (s *Service) remember(a Area) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.known == nil {
		s.known = map[string]bool{}
		for _, id := range strings.Split(s.KV.GetKV(areasKey), ",") {
			if id != "" {
				s.known[id] = true
			}
		}
	}
	if a.ID == "" || s.known[a.ID] {
		return
	}
	s.known[a.ID] = true
	var ids []string
	for id := range s.known {
		ids = append(ids, id)
	}
	_ = s.KV.SetKV(areasKey, strings.Join(ids, ","))
}

// RefreshKnown fetches again every area in use whose remembered prices are old. The EIA publishes once a week, so asking twice
// a day is plenty and costs one small request per area.
func (s *Service) RefreshKnown(ctx context.Context, key string) {
	s.remember(Area{}) // loads the list
	s.mu.Lock()
	var ids []string
	for id := range s.known {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		a, ok := areaByID(id)
		if !ok || id == "" {
			continue
		}
		if s.now().Sub(time.Unix(s.load(a).Fetched, 0)) > refreshAfter {
			_ = s.Refresh(ctx, key, a)
		}
	}
}
