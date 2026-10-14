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
	if e.BumpLevel == nil || trips.Driving(st) {
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
	m, has := Motion(st.Extra)
	if !has {
		return
	}
	e.S.NoteParkedReading(m, at)
	if m < candidateLevel {
		return
	}
	_ = e.S.AddParkedEvent(store.ParkedEvent{TS: at, Kind: "motion", Value: m, Note: fmt.Sprintf("%.2f m/s²", m)})
	if m >= level && !first && e.now().Sub(e.lastBump) >= bumpQuietFor {
		e.lastBump = e.now()
		e.send(push.Alert{Kind: push.BumpWhileParked, Title: "Bump while parked", Body: fmt.Sprintf("The parked car's motion sensors read %.1f m/s² (your level is %.1f). Take a look at the car.", m, level), URL: "#/parkedwatch"})
	}
}
