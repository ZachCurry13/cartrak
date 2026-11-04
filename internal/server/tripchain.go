package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/superchargers"
	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

const (
	superKey     = "superchargers"
	superAtKey   = "superchargers_at"
	superMaxAge  = 7 * 24 * time.Hour
	highwayKph   = 105.0
	chainStopMax = 4000
)

var superMu sync.Mutex

type chainIn struct {
	Start     stopPoint `json:"start"`
	Dest      stopPoint `json:"dest"`
	StartPct  float64   `json:"startPct"`
	Reserve   float64   `json:"reservePct"`
	Target    float64   `json:"targetPct"`
	ChargeKw  float64   `json:"chargeKw"`
	MaxLegKm  float64   `json:"maxLegKm"`
	MaxLegMin float64   `json:"maxLegMin"`
	DayMin    float64   `json:"dayMin"`
	Plugs     string    `json:"plugs"` // nacs, dock or both
}

// superSites is the Supercharger list kept in the database. It is downloaded again when it is over a week old (or missing); if
// that fails the old copy is used.
func (s *Server) superSites(ctx context.Context) (sites []superchargers.Site, at time.Time, err error) {
	superMu.Lock()
	defer superMu.Unlock()
	if b := s.S.GetKV(superKey); b != "" && json.Unmarshal([]byte(b), &sites) == nil {
		if n, _ := strconv.ParseInt(s.S.GetKV(superAtKey), 10, 64); n > 0 {
			at = time.Unix(n, 0)
		}
	}
	if len(sites) > 0 && time.Since(at) < superMaxAge {
		return sites, at, nil
	}
	url := s.SuperURL
	if url == "" {
		url = superchargers.DefaultURL
	}
	fctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	fresh, ferr := superchargers.Fetch(fctx, nil, url, "CarTrak/"+s.Version+" (+https://github.com/ZachCurry13/cartrak)")
	if ferr != nil || len(fresh) == 0 {
		if len(sites) > 0 {
			return sites, at, nil
		}
		if ferr == nil {
			ferr = errors.New("the Supercharger list was empty")
		}
		return nil, at, ferr
	}
	if b, merr := json.Marshal(fresh); merr == nil {
		_ = s.S.SetKV(superKey, string(b))
		_ = s.S.SetKV(superAtKey, strconv.FormatInt(time.Now().Unix(), 10))
	}
	return fresh, time.Now(), nil
}

// planChain: POST /api/trip-chain. Chooses Superchargers a non-Tesla car can use between a start and a destination, with no leg
// longer than the time or distance given, and cuts the trip into days. Admin only: the list may be downloaded from
// supercharge.info the first time and then weekly (nothing about the trip is sent).
func (s *Server) planChain(w http.ResponseWriter, r *http.Request) {
	var in chainIn
	bad := func(p stopPoint) bool {
		return p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 || (p.Lat == 0 && p.Lon == 0)
	}
	if decode(r, &in) != nil || bad(in.Start) || bad(in.Dest) || in.StartPct <= 0 || in.StartPct > 100 {
		fail(w, 400, "give a start, a destination and the starting charge")
		return
	}
	capacity := s.capacityKwh()
	base := s.whyBaseline()
	if s.fuelCar() || capacity == nil || base.Overall <= 0 {
		fail(w, 400, "Trip planning needs a battery size and some recorded trips first")
		return
	}
	if in.Reserve < 0 || in.Reserve > 60 {
		in.Reserve = 20
	}
	if in.Target < in.Reserve+10 || in.Target > 100 {
		in.Target = 90
	}
	if in.ChargeKw <= 0 || in.ChargeKw > 400 {
		in.ChargeKw = 100
	}
	if in.DayMin <= 0 || in.DayMin > 24*60 {
		in.DayMin = 12 * 60
	}
	kmPerKwh := base.ExpectedKmPerKwh(highwayKph, "")
	if kmPerKwh <= 0 {
		kmPerKwh = base.Overall
	}
	sites, at, err := s.superSites(r.Context())
	if err != nil {
		fail(w, upstreamFailed, "The Supercharger list could not be downloaded: "+err.Error())
		return
	}
	usable := superchargers.Usable(sites, in.Plugs)
	if len(usable) > chainStopMax {
		usable = usable[:chainStopMax]
	}
	stations := make([]tripchain.Station, len(usable))
	for i, u := range usable {
		stations[i] = tripchain.Station{Name: u.Name, Lat: u.Lat, Lon: u.Lon, Kw: u.Kw}
	}
	pt := func(p stopPoint, d string) tripchain.Station {
		if p.Name == "" {
			p.Name = d
		}
		return tripchain.Station{Name: p.Name, Lat: p.Lat, Lon: p.Lon}
	}
	days, err := tripchain.Plan(tripchain.Params{Start: pt(in.Start, "Start"), Dest: pt(in.Dest, "Destination"), StartPct: in.StartPct, ReservePct: in.Reserve,
		TargetPct: in.Target, CapacityKwh: *capacity, KwhPerKm: 1 / kmPerKwh, CarKw: in.ChargeKw, MaxLegKm: in.MaxLegKm, MaxLegMin: in.MaxLegMin,
		DayMin: in.DayMin, SpeedKph: highwayKph}, stations)
	if errors.Is(err, tripchain.ErrNoChain) {
		fail(w, http.StatusUnprocessableEntity, "No chain of Superchargers keeps every leg inside those limits. Allow longer legs, accept more plug types, or add a stop of your own.")
		return
	}
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	ok(w, map[string]any{"days": days, "sites": len(usable), "listAt": at.Unix()})
}
