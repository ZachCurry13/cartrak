package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/rebuild"
)

var repairMu sync.Mutex // one rebuild at a time

type repairIn struct {
	From    int64  `json:"from"`
	To      int64  `json:"to"`
	Confirm string `json:"confirm"`
}

func (s *Server) repairPlan(w http.ResponseWriter, r *http.Request) (*rebuild.Plan, repairIn, bool) {
	var in repairIn
	if decode(r, &in) != nil || in.From <= 0 || in.To <= 0 {
		fail(w, 400, "choose a start and an end")
		return nil, in, false
	}
	if s.D == nil {
		fail(w, 503, "trip detection is not available")
		return nil, in, false
	}
	p, err := rebuild.Compute(s.S, s.D, in.From, in.To, time.Now())
	if err != nil {
		fail(w, 500, "could not read the readings")
		return nil, in, false
	}
	p.Fill()
	return p, in, true
}

// repairPreview shows what rebuilding the trips and charging of a stretch of time would replace and what it would
// put there. It changes nothing. Admin only.
func (s *Server) repairPreview(w http.ResponseWriter, r *http.Request) {
	if p, _, ok2 := s.repairPlan(w, r); ok2 {
		ok(w, p)
	}
}

// repairApply does the rebuild for a stretch of time, after the owner typed REBUILD. Admin only.
func (s *Server) repairApply(w http.ResponseWriter, r *http.Request) {
	repairMu.Lock()
	defer repairMu.Unlock()
	p, in, ok2 := s.repairPlan(w, r)
	if !ok2 {
		return
	}
	if in.Confirm != "REBUILD" {
		fail(w, 400, "type REBUILD to confirm")
		return
	}
	if err := rebuild.Apply(s.S, s.D, p); err != nil {
		fail(w, 409, err.Error())
		return
	}
	loc := s.defaultLoc()
	s.S.LogDecision("fix a day", fmt.Sprintf("%s to %s: %d trips and %d charges replaced by %d and %d; %d trips and %d charges left alone (unchanged)",
		time.Unix(in.From, 0).In(loc).Format("Jan 2 15:04"), time.Unix(in.To, 0).In(loc).Format("Jan 2 15:04"), len(p.OldTrips), len(p.OldCharges), len(p.NewTrips), len(p.NewCharges), p.UnchangedTrips, p.UnchangedCharges))
	ok(w, map[string]any{"trips": len(p.NewTrips), "charges": len(p.NewCharges), "replacedTrips": len(p.OldTrips), "replacedCharges": len(p.OldCharges), "unchangedTrips": p.UnchangedTrips, "unchangedCharges": p.UnchangedCharges})
}
