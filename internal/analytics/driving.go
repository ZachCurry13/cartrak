package analytics

import (
	"sort"
	"time"
)

// Trip is one finished drive as the server sees it.
type Trip struct {
	ID        int64    `json:"id"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	Km        float64  `json:"km"`
	Kwh       float64  `json:"kwh"`                 // energy used, 0 when unknown
	Corrected bool     `json:"corrected,omitempty"` // its distance was adjusted by hand or from the estimated road, so it is not the odometer's own
	Gal       float64  `json:"gal,omitempty"`       // fuel burned, gallons, 0 when unknown (gas and hybrid cars)
	TempC     *float64 `json:"tempC,omitempty"`     // average outside temperature, when known
	MaxKph    float64  `json:"maxKph"`
	Category  string   `json:"category"`
}

// Point is one trip on the temperature chart.
type Point struct {
	ID       int64   `json:"id"`
	TempC    float64 `json:"tempC"`
	KmPerKwh float64 `json:"kmPerKwh"`
	Km       float64 `json:"km"`
}

// SpeedBin is how efficient trips were at an average speed.
type SpeedBin struct {
	FromKph  float64 `json:"fromKph"`
	ToKph    float64 `json:"toKph"`
	Trips    int     `json:"trips"`
	KmPerKwh float64 `json:"kmPerKwh"`
}

// LengthBin counts trips of a length.
type LengthBin struct {
	ToKm  float64 `json:"toKm"` // up to this many km; 0 for the last, open-ended bin
	Trips int     `json:"trips"`
}

// DayKm is how far the car went on a day.
type DayKm struct {
	Date  string  `json:"date"` // 2026-10-21, in the car's time zone
	Km    float64 `json:"km"`
	Trips int     `json:"trips"`
}

// DrivingReport is the Driving patterns page of Insights.
type DrivingReport struct {
	Trips    int            `json:"trips"`
	Km       float64        `json:"km"`
	Temp     []Point        `json:"temp"`    // trips with a temperature and energy
	TempFit  *[2]float64    `json:"tempFit"` // km/kWh = a + b·tempC over the temperature points, when there are enough
	Speed    []SpeedBin     `json:"speed"`   // by average speed
	Lengths  []LengthBin    `json:"lengths"`
	Days     []DayKm        `json:"days"`     // every day with driving, oldest first
	HourWeek [7][24]float64 `json:"hourWeek"` // km by weekday (Sunday first) and the hour the trip began
}

var lengthBins = []float64{2, 5, 10, 20, 50, 100, 0}
var speedEdges = []float64{0, 30, 50, 70, 90, 110, 200}

// Driving builds the report from finished trips.
func Driving(ts []Trip, loc *time.Location) DrivingReport {
	r := DrivingReport{}
	for _, b := range lengthBins {
		r.Lengths = append(r.Lengths, LengthBin{ToKm: b})
	}
	speed := make([][2]float64, len(speedEdges)-1) // km, kWh
	speedN := make([]int, len(speedEdges)-1)
	days := map[string]*DayKm{}
	for _, t := range ts {
		if t.Km < 0.3 || t.End <= t.Start {
			continue
		}
		r.Trips++
		r.Km += t.Km
		for i, b := range lengthBins {
			if b == 0 || t.Km < b {
				r.Lengths[i].Trips++
				break
			}
		}
		st := time.Unix(t.Start, 0).In(loc)
		r.HourWeek[int(st.Weekday())][st.Hour()] += t.Km
		key := st.Format("2006-01-02")
		d := days[key]
		if d == nil {
			d = &DayKm{Date: key}
			days[key] = d
		}
		d.Km += t.Km
		d.Trips++
		if t.Kwh <= 0 || t.Km < 2 {
			continue
		}
		eff := t.Km / t.Kwh
		if t.TempC != nil && eff > 0.5 && eff < 15 {
			r.Temp = append(r.Temp, Point{ID: t.ID, TempC: *t.TempC, KmPerKwh: eff, Km: t.Km})
		}
		avg := t.Km / (float64(t.End-t.Start) / 3600)
		for i := 0; i < len(speedEdges)-1; i++ {
			if avg >= speedEdges[i] && avg < speedEdges[i+1] {
				speed[i][0] += t.Km
				speed[i][1] += t.Kwh
				speedN[i]++
				break
			}
		}
	}
	for i, s := range speed {
		if speedN[i] > 0 && s[1] > 0 {
			r.Speed = append(r.Speed, SpeedBin{FromKph: speedEdges[i], ToKph: speedEdges[i+1], Trips: speedN[i], KmPerKwh: s[0] / s[1]})
		}
	}
	for _, d := range days {
		r.Days = append(r.Days, *d)
	}
	sort.Slice(r.Days, func(i, j int) bool { return r.Days[i].Date < r.Days[j].Date })
	r.TempFit = fit(r.Temp)
	return r
}

// fit is the straight line through the temperature points, each weighted by its distance, so a long drive counts for more
// than a hop to the shop. It needs at least eight trips across at least 8 °C.
func fit(ps []Point) *[2]float64 {
	if len(ps) < 8 {
		return nil
	}
	var sw, sx, sy, sxx, sxy, lo, hi float64
	lo, hi = ps[0].TempC, ps[0].TempC
	for _, p := range ps {
		w := p.Km
		sw += w
		sx += w * p.TempC
		sy += w * p.KmPerKwh
		sxx += w * p.TempC * p.TempC
		sxy += w * p.TempC * p.KmPerKwh
		lo, hi = min(lo, p.TempC), max(hi, p.TempC)
	}
	den := sw*sxx - sx*sx
	if hi-lo < 8 || den == 0 {
		return nil
	}
	b := (sw*sxy - sx*sy) / den
	a := (sy - b*sx) / sw
	return &[2]float64{a, b}
}
