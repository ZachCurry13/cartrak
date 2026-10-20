package alerts

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

const (
	bumpQuietFor   = 10 * time.Minute // one bump alert per stretch of shaking
	candidateLevel = 0.5              // readings this strong are listed on the Parked watch page even below the alert level
)

// motionVec is the car's three acceleration readings, or false when it reported none.
func motionVec(f map[string]any) ([3]float64, bool) {
	var v [3]float64
	any := false
	for i, k := range []string{"acceleration.value.x", "acceleration.value.y", "acceleration.value.z"} {
		if n := ford.Number(f, k); n != nil {
			v[i], any = *n, true
		}
	}
	return v, any
}

// Motion is how hard the car's motion sensors say it is being pushed (metres per second squared, gravity taken
// out), or false when the car reported none. A parked car reads close to zero.
func Motion(f map[string]any) (float64, bool) {
	x, y, z := ford.Number(f, "acceleration.value.x"), ford.Number(f, "acceleration.value.y"), ford.Number(f, "acceleration.value.z")
	if x == nil && y == nil && z == nil {
		return 0, false
	}
	sq := func(v *float64) float64 {
		if v == nil {
			return 0
		}
		return *v * *v
	}
	return math.Sqrt(sq(x) + sq(y) + sq(z)), true
}

// alarmGoingOff reads the alarm and panic-alarm states. Ford's exact words are not documented, so any state that
// says the alarm is active, triggered or sounding counts, and a state that says armed, disarmed, off or
// inactive does not.
func alarmGoingOff(f map[string]any) (string, bool) {
	for _, k := range []string{"alarmStatus", "panicAlarmStatus"} {
		v := strings.ToUpper(ford.Text(f, k))
		if v == "" {
			continue
		}
		if strings.Contains(v, "DISARM") || strings.Contains(v, "NOT_") || strings.Contains(v, "INACTIVE") || v == "OFF" || v == "ARMED" || v == "UNKNOWN" || v == "UNSUPPORTED" || v == "NORMAL" {
			continue
		}
		if strings.Contains(v, "ACTIV") || strings.Contains(v, "TRIGGER") || strings.Contains(v, "SOUND") || strings.Contains(v, "ALARMING") || v == "ON" {
			return k + " " + v, true
		}
	}
	return "", false
}

// parkedWatch looks at each reading of a parked car for a jolt or an alarm. Everything it sees above a low level
// is kept on the Parked watch page, so the owner can judge how well it works before relying on it.
func (e *Engine) parkedWatch(st *ford.State, first bool) {
	// Ignition on (even while the car charges) means someone is in it, so it is not "parked" for this purpose.
	if st.Driving() {
		e.lastDrive = st.TS
	}
	if e.BumpLevel == nil || st.Driving() {
		e.restVec = nil // the resting reading is learned again once the car is parked
		return
	}
	level := e.BumpLevel()
	if level <= 0 {
		return
	}
	at := st.TS.Unix()
	if note, bad := alarmGoingOff(st.Extra); bad && !first {
		_ = e.S.AddParkedEvent(store.ParkedEvent{TS: at, Kind: "alarm", Note: note})
		e.send(push.Alert{Kind: push.AlarmSounding, Title: "Car alarm", Body: "The car reports its alarm is going off (" + note + ").", URL: "#/parkedwatch"})
	}
	// A parked car's sensors rarely read exactly zero: a slope or a sensor offset gives a steady value. A bump is a
	// change from the reading just before, so a steady offset never counts.
	cur, has := motionVec(st.Extra)
	if !has {
		return
	}
	prev := e.restVec
	e.restVec = &cur
	if prev == nil {
		return
	}
	m := math.Sqrt(math.Pow(cur[0]-prev[0], 2) + math.Pow(cur[1]-prev[1], 2) + math.Pow(cur[2]-prev[2], 2))
	e.S.NoteParkedReading(m, at)
	if m < candidateLevel {
		return
	}
	_ = e.S.AddParkedEvent(store.ParkedEvent{TS: at, Kind: "motion", Value: m, Note: fmt.Sprintf("%.2f m/s²", m)})
	if m >= level && !first && e.now().Sub(e.lastBump) >= bumpQuietFor && st.TS.Sub(e.lastDrive) >= settleAfterDrive && !e.ownerWithCar(st) {
		e.lastBump = e.now()
		e.send(push.Alert{Kind: push.BumpWhileParked, Title: "Bump while parked", Body: fmt.Sprintf("The parked car's motion sensors read %.1f m/s² (your level is %.1f). Take a look at the car.", m, level), URL: "#/parkedwatch"})
	}
}

// settleAfterDrive is how long after the car was last in use a jump is not alerted: the door, the seat and the car settling,
// and the accelerometer reporting its first fresh value after the drive, look like a bump but are not.
const settleAfterDrive = 5 * time.Minute

// withCarRadiusM is how close a paired phone must be to the car for the owner to count as with it.
const withCarRadiusM = 150

// ownerWithCar says whether a bump is probably the owner: the car is unlocked (someone is getting in or out, loading
// it), or a paired phone reported a position beside the car in the last 15 minutes. Both are cases where a push
// about a bump would only be noise. The motion is still logged on the Parked watch page.
func (e *Engine) ownerWithCar(st *ford.State) bool {
	if st.Locked != nil && !*st.Locked {
		return true
	}
	if st.Lat == nil || st.Lon == nil {
		return false
	}
	pts, err := e.S.PhonePointsSince(e.now().Add(-15 * time.Minute).Unix())
	if err != nil {
		return false
	}
	for _, p := range pts {
		if trips.Haversine(*st.Lat, *st.Lon, p[0], p[1]) <= withCarRadiusM {
			return true
		}
	}
	return false
}
