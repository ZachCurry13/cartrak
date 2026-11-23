package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/insights"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/routeplan"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	maxRouteStops    = 11 // the start and up to ten destinations
	straightLineBend = 1.3
	straightLineKph  = 65.0
)

type routePlanIn struct {
	Stops     []stopPoint `json:"stops"`
	DepartTS  int64       `json:"departTs"` // unix seconds; 0 is now
	StartPct  float64     `json:"startPct"`
	Reserve   float64     `json:"reservePct"`
	Target    float64     `json:"targetPct"`
	ChargeKw  float64     `json:"chargeKw"`
	Optimize  bool        `json:"optimize"`  // put the destinations in the order that takes least driving
	RoundTrip bool        `json:"roundTrip"` // and drive back to the start at the end
	EndLast   bool        `json:"endLast"`   // without the way back: the last stop given stays last (Costco on the way home)
	StayMin   float64     `json:"stayMin"`   // minutes spent at each stop that does not say its own
}

type routeLegOut struct {
	routeplan.LegOut
	Weather   legWeather `json:"weather"`
	DepartTS  int64      `json:"departTs"`
	ArriveTS  int64      `json:"arriveTs"`
	Estimated bool       `json:"estimated"` // the distance is a straight line made longer, not a road
	At        stopTime   `json:"at"`        // at the destination: the wait for its set time, how late, the time spent there
}

// planRoute: POST /api/route-plan. Works the battery through up to ten destinations, with the forecast for each leg and how the
// car usually drives in that weather. Admin only, because it asks the road and weather services about the places given.
func (s *Server) planRoute(w http.ResponseWriter, r *http.Request) {
	var in routePlanIn
	if decode(r, &in) != nil || len(in.Stops) < 2 || len(in.Stops) > maxRouteStops {
		fail(w, 400, "give a start and one to ten destinations")
		return
	}
	for i := range in.Stops {
		p := &in.Stops[i]
		if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 || (p.Lat == 0 && p.Lon == 0) {
			fail(w, 400, "CarTrak does not know where one of the stops is")
			return
		}
		if len(p.Name) > 80 {
			p.Name = p.Name[:80]
		}
	}
	if in.StartPct <= 0 || in.StartPct > 100 {
		fail(w, 400, "give the starting charge, 1 to 100")
		return
	}
	if in.Reserve < 0 || in.Reserve > 60 {
		in.Reserve = 20
	}
	if in.Target < in.Reserve+10 || in.Target > 100 {
		in.Target = 90
	}
	if in.StayMin < 0 || in.StayMin > maxStayMin {
		in.StayMin = 0
	}
	if in.ChargeKw <= 0 || in.ChargeKw > 400 {
		in.ChargeKw = 100
	}
	capacity := s.capacityKwh()
	base := s.whyBaseline()
	if s.fuelCar() || capacity == nil || base.Overall <= 0 {
		fail(w, 400, "Route planning needs a battery size and some recorded trips first")
		return
	}
	depart := in.DepartTS
	if depart <= 0 {
		depart = time.Now().Unix()
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	var fc *forecast
	if s.Cfg != nil && s.Cfg.WeatherLookup() {
		fc = s.newForecast(ctx)
	}
	roads := s.routeLookupOn()
	notes := []string{}
	order, saved := make([]int, len(in.Stops)), 0.0
	for i := range order {
		order[i] = i
	}
	v := visits(in.Stops, depart, in.StayMin)
	onTime := true
	if in.Optimize && len(in.Stops) >= 3 {
		in.Stops, order, saved, onTime = s.timedOrder(ctx, roads, in.Stops, v, in.RoundTrip, in.EndLast)
		v = visits(in.Stops, depart, in.StayMin)
	}
	if in.RoundTrip {
		home := in.Stops[0]
		home.AtTS, home.StayMin = 0, nil
		in.Stops = append(in.Stops, home)
		v = append(v, routeplan.Visit{AtMin: routeplan.NoTime})
	}
	var legs []routeplan.Leg
	var wx []legWeather
	var est []bool
	at := depart
	for i := 1; i < len(in.Stops); i++ {
		a, b := in.Stops[i-1], in.Stops[i]
		km, minutes, estimated := s.legRoad(ctx, roads, a, b)
		lw, factor := s.weatherForLeg(fc, a, b, at, km, minutes)
		group := ""
		if lw.TempC != nil {
			group = lw.TempBand
		} else if x, _ := s.S.LatestSample(); x != nil && x.OutsideC != nil {
			group = insights.BandName(*x.OutsideC) // no forecast: the temperature outside now
		}
		kph := km / minutes * 60
		exp := base.ExpectedKmPerKwh(kph, group)
		if exp <= 0 {
			exp = base.Overall
		}
		if factor == 0 {
			factor = 1
		}
		legs = append(legs, routeplan.Leg{From: stopName(a, i), To: stopName(b, i+1), Km: km, Minutes: minutes, KwhPerKm: factor / exp})
		wx, est = append(wx, lw), append(est, estimated)
		at, _ = arriveAt(v[i], in.Stops[i], depart, at+int64(minutes*60)) // the forecast is for when each leg is really driven
	}
	plan := routeplan.Plan(routeplan.Input{Legs: legs, StartPct: in.StartPct, ReservePct: in.Reserve, TargetPct: in.Target, CapacityKwh: *capacity, ChargeKw: in.ChargeKw})
	out := make([]routeLegOut, len(plan.Legs))
	t := depart
	anyEst, anyFc := false, false
	var late []string
	for i, l := range plan.Legs {
		if l.Charge != nil {
			t += int64(l.Charge.Minutes * 60)
		}
		out[i] = routeLegOut{LegOut: l, Weather: wx[i], DepartTS: t, Estimated: est[i]}
		t += int64(l.Minutes * 60)
		out[i].ArriveTS = t
		t, out[i].At = arriveAt(v[i+1], in.Stops[i+1], depart, t)
		if out[i].At.LateMin > routeplan.OnTimeSlack {
			late = append(late, lateWarning(stopName(in.Stops[i+1], i+1), out[i].At, s.defaultLoc()))
		}
		anyEst = anyEst || est[i]
		anyFc = anyFc || wx[i].Known
	}
	if anyEst {
		notes = append(notes, "Some distances are a straight line made 30% longer, because the road lookup is off or did not answer. Turn on Road estimates in Settings for real roads.")
	}
	if !anyFc {
		notes = append(notes, "No forecast: turn on the Weather lookup in Settings so each leg uses the weather expected on it. Until then the temperature outside now is used.")
	}
	ok(w, map[string]any{"legs": out, "totalKm": plan.TotalKm, "drivingMinutes": plan.DrivingMinutes, "chargingMinutes": plan.ChargingMinutes,
		"totalKwh": plan.TotalKwh, "arrivePct": plan.ArrivePct, "warnings": append(append([]string{}, late...), plan.Warnings...), "onTime": onTime && len(late) == 0, "stayMin": in.StayMin, "endLast": in.EndLast, "notes": notes, "stops": in.Stops, "order": order, "savedMinutes": saved, "roundTrip": in.RoundTrip, "reservePct": in.Reserve, "targetPct": in.Target, "departTs": depart})
}

func stopName(p stopPoint, n int) string {
	if p.Name != "" {
		return p.Name
	}
	return "Stop " + strconv.Itoa(n)
}

// legRoad is the road distance and driving time between two stops, or a straight line made longer when the road lookup is off or
// cannot answer.
func (s *Server) legRoad(ctx context.Context, roads bool, a, b stopPoint) (km, minutes float64, estimated bool) {
	if roads {
		if k, m, err := s.routeClient().Drive(ctx, []routeest.Point{{a.Lat, a.Lon}, {b.Lat, b.Lon}}); err == nil && k > 0 && m > 0 {
			time.Sleep(250 * time.Millisecond) // the free routing service asks for a gentle pace
			return k, m, false
		}
	}
	km = trips.Haversine(a.Lat, a.Lon, b.Lat, b.Lon) / 1000 * straightLineBend
	if km < 0.2 {
		km = 0.2
	}
	return km, km / straightLineKph * 60, true
}
