// Package tirewatch follows tire pressures over days and looks for a tire that is slowly going down. Pressure
// follows temperature, so each reading is first scaled to 20 degrees C using the outside temperature.
package tirewatch

import "sort"

var Names = [4]string{"Front left", "Front right", "Rear left", "Rear right"}

// Point is one reading of the four tires (kilopascals) and the outside temperature.
type Point struct {
	TS       int64
	P        [4]*float64
	OutsideC *float64
}

// Tire is what is known about one tire.
type Tire struct {
	Name     string   `json:"name"`
	Kpa      *float64 `json:"kpa"`      // the latest reading
	AtTwenty *float64 `json:"atTwenty"` // scaled to 20 C, comparable between days
	PerDay   *float64 `json:"perDay"`   // kPa per day over the last two weeks, scaled the same way
	Leak     bool     `json:"leak"`
}

// Report is the four tires and the placard values the car reports.
type Report struct {
	Tires        [4]Tire  `json:"tires"`
	PlacardFront *float64 `json:"placardFront"`
	PlacardRear  *float64 `json:"placardRear"`
	Days         int      `json:"days"` // days of readings the trend rests on
	Note         string   `json:"note"`
}

const (
	leakKpaPerDay   = -1.5 // about 0.2 psi a day
	othersKpaPerDay = -0.6 // the other tires are steady
	minDays         = 5
)

func scale(p float64, outsideC *float64) float64 {
	if outsideC == nil {
		return p
	}
	return p * 293.15 / (*outsideC + 273.15)
}

// Build looks at readings oldest first.
func Build(points []Point, placardFront, placardRear *float64) Report {
	r := Report{PlacardFront: placardFront, PlacardRear: placardRear}
	for i := range r.Tires {
		r.Tires[i].Name = Names[i]
	}
	sort.Slice(points, func(i, j int) bool { return points[i].TS < points[j].TS })
	type dayAvg struct{ x, y float64 }
	series := [4][]dayAvg{}
	byDay := map[int64]*[4][]float64{}
	var order []int64
	for _, pt := range points {
		day := pt.TS / 86400
		d := byDay[day]
		if d == nil {
			d = &[4][]float64{}
			byDay[day] = d
			order = append(order, day)
		}
		for i := 0; i < 4; i++ {
			if pt.P[i] != nil && *pt.P[i] > 50 {
				d[i] = append(d[i], scale(*pt.P[i], pt.OutsideC))
			}
		}
	}
	for _, day := range order {
		for i := 0; i < 4; i++ {
			if v := byDay[day][i]; len(v) > 0 {
				var s float64
				for _, x := range v {
					s += x
				}
				series[i] = append(series[i], dayAvg{float64(day), s / float64(len(v))})
			}
		}
	}
	if n := len(points); n > 0 {
		last := points[n-1]
		for i := 0; i < 4; i++ {
			if last.P[i] != nil {
				k := *last.P[i]
				r.Tires[i].Kpa = &k
				a := scale(k, last.OutsideC)
				r.Tires[i].AtTwenty = &a
			}
		}
	}
	r.Days = len(order)
	slopes := [4]*float64{}
	for i := 0; i < 4; i++ {
		s := series[i]
		if len(s) < minDays {
			continue
		}
		var mx, my float64
		for _, p := range s {
			mx += p.x
			my += p.y
		}
		mx /= float64(len(s))
		my /= float64(len(s))
		var num, den float64
		for _, p := range s {
			num += (p.x - mx) * (p.y - my)
			den += (p.x - mx) * (p.x - mx)
		}
		if den == 0 {
			continue
		}
		sl := num / den
		slopes[i] = &sl
		r.Tires[i].PerDay = &sl
	}
	for i := 0; i < 4; i++ {
		if slopes[i] == nil || *slopes[i] > leakKpaPerDay {
			continue
		}
		steady, others := 0, 0
		for j := 0; j < 4; j++ {
			if j != i && slopes[j] != nil {
				others++
				if *slopes[j] >= othersKpaPerDay {
					steady++
				}
			}
		}
		if others >= 2 && steady == others {
			r.Tires[i].Leak = true
			r.Note = r.Tires[i].Name + " is losing pressure faster than the others, even allowing for the temperature. Check it for a nail or a leaking valve."
		}
	}
	return r
}
