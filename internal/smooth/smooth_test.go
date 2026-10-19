package smooth

import "testing"

func TestJudgeAgainstTheDriversOwnFirmLevel(t *testing.T) {
	var recent []float64
	for i := 1; i <= 100; i++ {
		recent = append(recent, float64(i))
	}
	p := Percentile90(recent)
	if p < 89 || p > 91 {
		t.Fatalf("p90: %v", p)
	}
	if Percentile90([]float64{5, 6}) != 0 || Percentile90([]float64{-3, 0, 0}) != 0 {
		t.Fatal("too little to say")
	}
	b := Baseline{Accel: 90, Brake: 50}
	gentle := make([]Point, 20)
	for i := range gentle {
		gentle[i] = Point{Accel: 30, Brake: 5}
	}
	if r, ok := Judge(gentle, b); !ok || r.Verdict != "smooth" || r.AccelShare != 0 {
		t.Fatalf("gentle: %+v", r)
	}
	hard := make([]Point, 20)
	for i := range hard {
		hard[i] = Point{Accel: 120, Brake: 60}
		if i%4 == 3 {
			hard[i] = Point{Accel: 20}
		}
	}
	if r, ok := Judge(hard, b); !ok || r.Verdict != "firm" || r.Peak != 120 {
		t.Fatalf("hard: %+v", r)
	}
	if _, ok := Judge(hard[:3], b); ok {
		t.Fatal("a trip of three readings is not judged")
	}
	if _, ok := Judge(hard, Baseline{}); ok {
		t.Fatal("no baseline, no judgement")
	}
}
