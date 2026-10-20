package server

import (
	"encoding/json"
	"sync"
	"time"
)

// kvAdminViews holds, per car, which administrators opened it without having been given access, and when last.
const kvAdminViews = "admin_views"

var adminViewMu sync.Mutex

// noteAdminView records that an administrator opened this car. It is kept in the car's own database so its owner can be
// told, and written at most once every ten minutes for the same administrator.
func (s *Server) noteAdminView(name string) {
	adminViewMu.Lock()
	defer adminViewMu.Unlock()
	m := ReadAdminViews(s.S.GetKV(kvAdminViews))
	now := time.Now().Unix()
	if now-m[name] < 600 {
		return
	}
	m[name] = now
	b, _ := json.Marshal(m)
	_ = s.S.SetKV(kvAdminViews, string(b))
}

// ReadAdminViews decodes the stored record.
func ReadAdminViews(raw string) map[string]int64 {
	m := map[string]int64{}
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

// AdminViewsKey is the key the record is stored under, for the fleet to read another car's.
const AdminViewsKey = kvAdminViews
