// Package remotestart counts remote starts from the countdown the car reports while one is running. The count is
// CarTrak's own, worked out from the readings it saw: a start that began and ended between two readings is missed.
package remotestart

// Point is the countdown (seconds left) in one reading. Zero or less means no remote start is running.
type Point struct {
	TS   int64
	Left float64
}

// Count is the number of remote starts that began in points (oldest first). A start begins when the countdown rises
// from nothing, or when it jumps up while running (the car restarted it).
func Count(points []Point) int {
	n := 0
	prev := 0.0
	for _, p := range points {
		if p.Left > 0 && (prev <= 0 || p.Left > prev+60) {
			n++
		}
		prev = p.Left
	}
	return n
}
