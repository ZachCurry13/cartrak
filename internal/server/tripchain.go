package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ZachCurry13/cartrak/internal/superchargers"
	"github.com/ZachCurry13/cartrak/internal/tripchain"
)

const (
	highwayKph             = 105.0
	defaultPaceKph         = 8.0 // about 5 mph over
	chainStopMax           = 6000
	altMinSaving           = 5.0     // minutes: a smaller saving is not worth mentioning
	eaPenaltyMin           = 10.0    // Superchargers first: Electrify America is used when it saves more than this
	viaSnapM       float64 = 25000.0 // "find a charger here": the closest usable charger within this many meters of the place
)

type chainVia struct {
	stopPoint
	Mode   string `json:"mode"`   // pass (drive through), night (a hotel) or charge (find a charger here)
	Charge *bool  `json:"charge"` // a hotel has a charger; in older pages, plug in here
	Night  bool   `json:"night"`  // the day ends here: a hotel already booked
}

type chainIn struct {
	Start     stopPoint  `json:"start"`
	Dest      stopPoint  `json:"dest"`
	Via       []chainVia `json:"via"` // places to pass, in order
	StartPct  float64    `json:"startPct"`
	Reserve   float64    `json:"reservePct"`
	Target    float64    `json:"targetPct"`
	ChargeKw  float64    `json:"chargeKw"`
	MaxLegKm  float64    `json:"maxLegKm"`
	MaxLegMin float64    `json:"maxLegMin"`
	DayMin    float64    `json:"dayMin"`
	Plugs     string     `json:"plugs"`    // Tesla Superchargers: nacs, dock, both, or none
	EA        bool       `json:"ea"`       // also use Electrify America
	PaceKph   *float64   `json:"paceKph"`  // how much faster than the road's own speed you drive on highways; unset is 5 mph
	Avoid     []string   `json:"avoid"`    // chargers never to use, as "lat,lon" to four places
	DepartTS  int64      `json:"departTs"` // when the trip leaves; 0 is the next morning at eight
}

func chargerKey(lat, lon float64) string { return fmt.Sprintf("%.4f,%.4f", lat, lon) }

func total(days []tripchain.Day) (min float64, stops int) {
	for _, d := range days {
		min += d.DriveMin + d.ChargeMin
		stops += d.ChargeStops
	}
	return
}

// planChain: POST /api/trip-chain. Chooses chargers a non-Tesla car can use between a start and a destination, through the
// places given, with no leg longer than the time or distance given, and cuts the trip into days. Superchargers come first;
// Electrify America is added when asked. Admin only: the lists may be downloaded the first time and then weekly (nothing about
// the trip is sent).
func (s *Server) planChain(w http.ResponseWriter, r *http.Request) {
	var in chainIn
	bad := func(p stopPoint) bool {
		return p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 || (p.Lat == 0 && p.Lon == 0)
	}
	if decode(r, &in) != nil || bad(in.Start) || bad(in.Dest) || in.StartPct <= 0 || in.StartPct > 100 || len(in.Via) > 6 || len(in.Avoid) > 200 {
		fail(w, 400, "give a start, a destination, up to six places to pass and the starting charge")
		return
	}
	for _, v := range in.Via {
		if bad(v.stopPoint) {
			fail(w, 400, "CarTrak does not know where one of the places to pass is")
			return
		}
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
	pace := defaultPaceKph
	if in.PaceKph != nil && *in.PaceKph >= 0 && *in.PaceKph <= 40 {
		pace = *in.PaceKph
	}
	energyAt := func(kph float64) float64 { // kWh per km at an average speed, from how this car has driven at it
		v := base.ExpectedKmPerKwh(kph, "")
		if v <= 0 {
			v = base.Overall
		}
		return 1 / v
	}
	kmPerKwh := 1 / energyAt(highwayKph+pace) // for a leg not yet checked on a real road
	var tesla, ea []superchargers.Site
	if in.Plugs != "none" {
		all, _, err := s.superSites(r.Context())
		if err != nil {
			fail(w, upstreamFailed, "The Supercharger list could not be downloaded: "+err.Error())
			return
		}
		tesla = superchargers.Usable(all, in.Plugs)
	}
	key := ""
	if s.Cfg != nil {
		key = s.Cfg.AFDCKey()
	}
	useEA := in.EA || in.Plugs == "none"
	if useEA && key == "" {
		fail(w, 400, "Electrify America needs the free charger list key: paste it in Settings (Charger list key)")
		return
	}
	if key != "" {
		var err error
		if ea, err = s.eaSites(r.Context(), key); err != nil && useEA {
			fail(w, upstreamFailed, "The Electrify America list could not be downloaded: "+err.Error())
			return
		}
	}
	avoid := map[string]bool{}
	for _, k := range in.Avoid {
		avoid[k] = true
	}
	station := func(p stopPoint, d string) tripchain.Station {
		if p.Name == "" {
			p.Name = d
		}
		return tripchain.Station{Name: p.Name, Lat: p.Lat, Lon: p.Lon}
	}
	use := [][]superchargers.Site{tesla}
	if useEA {
		use = append(use, ea)
	}
	var eligible []superchargers.Site // the chargers a place may be snapped to: the networks in use, minus the ones avoided
	for _, l := range use {
		for _, u := range l {
			if !avoid[chargerKey(u.Lat, u.Lon)] {
				eligible = append(eligible, u)
			}
		}
	}
	vias := make([]tripchain.Via, len(in.Via))
	for i, v := range in.Via {
		at := station(v.stopPoint, "Stop")
		switch v.Mode {
		case "charge": // find a charger here: the stop becomes the closest usable charger
			site, found := superchargers.Nearest(eligible, v.Lat, v.Lon, viaSnapM)
			if !found {
				fail(w, http.StatusUnprocessableEntity, fmt.Sprintf("No charger you can use is within %.0f miles of %s. Pick a bigger town, or another way to treat that place.", viaSnapM/1609.0, at.Name))
				return
			}
			name := site.Name
			if site.Net == "" {
				name = "Supercharger: " + name
			}
			vias[i] = tripchain.Via{Station: tripchain.Station{Name: name, Lat: site.Lat, Lon: site.Lon, Kw: site.Kw, PS: site.PS, Net: site.Net}, Charge: true}
		case "night": // a hotel: the day ends here; it charges only if it has a charger
			vias[i] = tripchain.Via{Station: at, Night: true, Charge: v.Charge != nil && *v.Charge}
		case "pass":
			vias[i] = tripchain.Via{Station: at}
		default: // older pages: the two flags
			vias[i] = tripchain.Via{Station: at, Charge: v.Charge == nil || *v.Charge, Night: v.Night}
		}
	}
	params := tripchain.Params{Start: station(in.Start, "Start"), Dest: station(in.Dest, "Destination"), StartPct: in.StartPct, ReservePct: in.Reserve,
		TargetPct: in.Target, CapacityKwh: *capacity, KwhPerKm: 1 / kmPerKwh, CarKw: in.ChargeKw, MaxLegKm: in.MaxLegKm, MaxLegMin: in.MaxLegMin,
		DayMin: in.DayMin, SpeedKph: highwayKph + pace}
	rl := newRoadLegs(r.Context(), s) // the real roads of the legs, asked as the plan is made
	rl.paceKph, rl.energyAt = pace, energyAt
	plan := func(lists ...[]superchargers.Site) ([]tripchain.Day, int, error) {
		var st []tripchain.Station
		for _, list := range lists {
			for _, u := range list {
				if avoid[chargerKey(u.Lat, u.Lon)] {
					continue
				}
				x := tripchain.Station{Name: u.Name, Lat: u.Lat, Lon: u.Lon, Kw: u.Kw, PS: u.PS, Net: u.Net}
				if u.Net != "" {
					x.Penalty = eaPenaltyMin
				}
				st = append(st, x)
			}
		}
		if len(st) > chainStopMax {
			st = st[:chainStopMax]
		}
		days, err := rl.plan(params, st, vias)
		return days, len(st), err
	}
	days, n, err := plan(use...)
	if errors.Is(err, tripchain.ErrNoChain) {
		fail(w, http.StatusUnprocessableEntity, "No chain of chargers keeps every leg inside those limits. Allow longer legs, accept more plug types or networks, or change a place to pass.")
		return
	}
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	s.addNearby(days, use, avoid)
	prices := s.fillTimesAndCosts(days, in.DepartTS)
	out := map[string]any{"days": days, "sites": n, "prices": prices, "currency": s.currency(), "exact": rl.exact(days), "roads": rl.on}
	if !useEA && len(ea) > 0 { // what adding Electrify America would change
		if alt, _, aerr := plan(tesla, ea); aerr == nil {
			m0, s0 := total(days)
			m1, s1 := total(alt)
			if m0-m1 >= altMinSaving {
				out["alt"] = map[string]any{"minutesSaved": m0 - m1, "chargeStops": s1, "was": s0}
			}
		}
	}
	ok(w, out)
}
