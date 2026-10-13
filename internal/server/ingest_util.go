package server

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// num reads a JSON number or a quoted number, because automation apps often build
// the body by hand and put quotes around everything.
type num float64

func (n *num) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("not a number")
	}
	*n = num(f)
	return nil
}

func ptrf(n *num) *float64 {
	if n == nil {
		return nil
	}
	f := float64(*n)
	return &f
}

// rateWindow limits each device to callsPerMinute calls a minute.
type rateWindow struct {
	mu   sync.Mutex
	seen map[int64][]time.Time
}

func (r *rateWindow) allow(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen == nil {
		r.seen = map[int64][]time.Time{}
	}
	cut := time.Now().Add(-time.Minute)
	keep := r.seen[id][:0]
	for _, t := range r.seen[id] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	r.seen[id] = keep
	if len(keep) >= callsPerMinute {
		return false
	}
	r.seen[id] = append(keep, time.Now())
	return true
}
