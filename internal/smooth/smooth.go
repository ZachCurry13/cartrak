// Package smooth judges how firmly a trip was driven from the car's torque readings, against the driver's own recent
// trips. The torque fields have no stated unit, so nothing here uses an absolute number: a reading counts as firm when
// it is above what the driver's own recent trips reached nine times out of ten.
package smooth

import "sort"

// Point is one reading while the car was moving.
type Point struct {
	Accel float64 // torque pushing the car, zero or more
	Brake float64 // braking torque, zero or more
}

// Baseline is what firm means for this driver.
type Baseline struct {
	Accel, Brake float64 // the 90th percentile of the readings above zero in recent trips
	Trips        int
}

// Result is how firm a trip was.
type Result struct {
	AccelShare float64 `json:"accelShare"` // percent of moving readings above the driver's firm level
	BrakeShare float64 `json:"brakeShare"`
	Peak       float64 `json:"peakTorque"`
	Verdict    string  `json:"verdict"` // smooth, usual, firm
	Moving     int     `json:"readings"`
}

// Percentile90 is the 90th percentile of the values above zero, or 0 when there are too few to say.
func Percentile90(vals []float64) float64 {
	var v []float64
	for _, x := range vals {
		if x > 0 {
			v = append(v, x)
		}
	}
	if len(v) < 20 {
		return 0
	}
	sort.Float64s(v)
	return v[int(float64(len(v)-1)*0.9)]
}

// Judge compares one trip with the baseline. ok is false when there is no baseline or too little of the trip to judge.
func Judge(pts []Point, b Baseline) (Result, bool) {
	var r Result
	if b.Accel <= 0 || len(pts) < 8 {
		return r, false
	}
	var a, br int
	for _, p := range pts {
		if p.Accel > r.Peak {
			r.Peak = p.Accel
		}
		if p.Accel >= b.Accel {
			a++
		}
		if b.Brake > 0 && p.Brake >= b.Brake {
			br++
		}
	}
	r.Moving = len(pts)
	r.AccelShare = float64(a) / float64(len(pts)) * 100
	r.BrakeShare = float64(br) / float64(len(pts)) * 100
	switch firm := r.AccelShare + r.BrakeShare; {
	case firm >= 28: // about ten percent each is usual by definition; well above that is firm
		r.Verdict = "firm"
	case firm <= 8:
		r.Verdict = "smooth"
	default:
		r.Verdict = "usual"
	}
	return r, true
}
