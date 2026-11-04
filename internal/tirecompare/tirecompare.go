// Package tirecompare answers "do my winter tires change my range?" without blaming the tires for the weather. Cold air costs energy
// by itself, and winter tires go on when it turns cold, so an overall before-and-after would measure the season. Instead the trips on
// winter tires and on other tires are put in cells by outside temperature and by speed, and only cells that both kinds of tire have
// driven enough in are compared. With too little overlap it says so and gives no number.
package tirecompare

import (
	"math"
	"sort"
)

const (
	CellC      = 5.0   // width of a temperature cell, degrees C (about 9 F)
	HighwayKph = 72.0  // trips averaging at least this are "highway" (45 mph), the rest "city"
	minCellKm  = 30.0  // each side needs this much driving in a cell for the cell to count
	minTrips   = 2     // and this many trips
	okKm       = 150.0 // matched distance (the smaller side of every cell, added up) that makes the answer "fair"
	goodKm     = 400.0 // and "good"
	minTripKm  = 3.0   // shorter trips say nothing about efficiency
	minTripKwh = 0.5
)

// Trip is one finished trip with the numbers the comparison needs.
type Trip struct {
	ID     int64
	Km     float64
	Kwh    float64 // energy used
	TempC  float64
	HasT   bool
	AvgKph float64
	Wet    bool // rain or snow on the road (left out)
	Winter bool // was on winter tires
}

// Side is one kind of tire's driving in a cell.
type Side struct {
	Km    float64 `json:"km"`
	Kwh   float64 `json:"kwh"`
	Trips int     `json:"trips"`
}

// KmPerKwh is the side's efficiency, 0 when there is no energy.
func (s Side) KmPerKwh() float64 {
	if s.Kwh <= 0 {
		return 0
	}
	return s.Km / s.Kwh
}

// Cell is one temperature and speed class with both sides.
type Cell struct {
	LoC     float64 `json:"loC"`
	HiC     float64 `json:"hiC"`
	Highway bool    `json:"highway"`
	Winter  Side    `json:"winter"`
	Other   Side    `json:"other"`
	Matched bool    `json:"matched"` // both sides have enough here to be compared
	Pct     float64 `json:"pct"`     // winter's energy per km against the other's, in percent (positive: winter tires use more)
}

// Result is the comparison.
type Result struct {
	Cells     []Cell  `json:"cells"`
	Matched   int     `json:"matched"`   // cells compared
	MatchedKm float64 `json:"matchedKm"` // distance the comparison rests on (the smaller side of each cell)
	Pct       float64 `json:"pct"`       // overall, weighted by each cell's matched distance
	LowPct    float64 `json:"lowPct"`    // the smallest and largest matched cell, so the spread is visible
	HighPct   float64 `json:"highPct"`
	Verdict   string  `json:"verdict"` // "none" (not enough overlap), "fair" or "good"
	WinterKm  float64 `json:"winterKm"`
	OtherKm   float64 `json:"otherKm"`
	Excluded  int     `json:"excluded"` // trips left out for rain or snow
}

func band(c float64) (lo, hi float64) {
	lo = math.Floor(c/CellC) * CellC
	return lo, lo + CellC
}

// Compare builds the cells and the answer.
func Compare(trips []Trip) Result {
	type key struct {
		lo      float64
		highway bool
	}
	cells := map[key]*Cell{}
	var res Result
	for _, t := range trips {
		if !t.HasT || t.Km < minTripKm || t.Kwh < minTripKwh || t.AvgKph <= 0 {
			continue
		}
		if t.Wet {
			res.Excluded++
			continue
		}
		lo, hi := band(t.TempC)
		k := key{lo, t.AvgKph >= HighwayKph}
		c := cells[k]
		if c == nil {
			c = &Cell{LoC: lo, HiC: hi, Highway: k.highway}
			cells[k] = c
		}
		s := &c.Other
		if t.Winter {
			s = &c.Winter
			res.WinterKm += t.Km
		} else {
			res.OtherKm += t.Km
		}
		s.Km += t.Km
		s.Kwh += t.Kwh
		s.Trips++
	}
	var wsum, wpct float64
	res.LowPct, res.HighPct = math.Inf(1), math.Inf(-1)
	for _, c := range cells {
		if c.Winter.Km >= minCellKm && c.Other.Km >= minCellKm && c.Winter.Trips >= minTrips && c.Other.Trips >= minTrips {
			c.Matched = true
			c.Pct = (c.Winter.Kwh/c.Winter.Km)/(c.Other.Kwh/c.Other.Km)*100 - 100
			w := math.Min(c.Winter.Km, c.Other.Km)
			wsum += w
			wpct += w * c.Pct
			res.Matched++
			res.LowPct, res.HighPct = math.Min(res.LowPct, c.Pct), math.Max(res.HighPct, c.Pct)
		}
		res.Cells = append(res.Cells, *c)
	}
	sort.Slice(res.Cells, func(i, j int) bool {
		if res.Cells[i].LoC != res.Cells[j].LoC {
			return res.Cells[i].LoC < res.Cells[j].LoC
		}
		return !res.Cells[i].Highway && res.Cells[j].Highway
	})
	if res.Cells == nil {
		res.Cells = []Cell{}
	}
	res.MatchedKm = wsum
	switch {
	case res.Matched == 0 || wsum < okKm:
		res.Verdict = "none"
		res.LowPct, res.HighPct = 0, 0
	case wsum >= goodKm && res.Matched >= 3:
		res.Verdict, res.Pct = "good", wpct/wsum
	default:
		res.Verdict, res.Pct = "fair", wpct/wsum
	}
	return res
}
