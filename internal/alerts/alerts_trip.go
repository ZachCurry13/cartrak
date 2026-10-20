package alerts

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/push"
)

// trip announces a trip that has just begun or ended, from the trip detector's records.
func (e *Engine) trip(first bool) {
	open, _ := e.S.OpenTrip()
	var id int64
	// A trip is announced once it has actually gone somewhere. One that is only sitting with the ignition on, or that is
	// about to be thrown away as a blip, would otherwise send a "started" now and an "ended" later for nothing.
	if open != nil && (open.DistanceKm >= 0.3 || open.MaxSpeedKph >= 8) {
		id = open.ID
	} else if open != nil {
		id = e.openTrip // not announced yet: keep what is already announced
	}
	if !first && id != e.openTrip {
		if id != 0 && e.openTrip == 0 {
			e.send(push.Alert{Kind: push.TripStart, Title: "Trip started", Body: "CarTrak is recording a trip.", URL: "#/trip/" + strconv.FormatInt(id, 10)})
		}
		if e.openTrip != 0 {
			if t, err := e.S.GetTrip(e.openTrip); err == nil && !t.Open {
				e.send(push.Alert{Kind: push.TripEnd, Title: "Trip ended", Body: fmt.Sprintf("%.1f mi (%.1f km).", t.DistanceKm*0.621371, t.DistanceKm), URL: "#/trip/" + strconv.FormatInt(t.ID, 10)})
			}
			if id != 0 {
				e.send(push.Alert{Kind: push.TripStart, Title: "Trip started", Body: "CarTrak is recording a trip.", URL: "#/trip/" + strconv.FormatInt(id, 10)})
			}
		}
	}
	e.openTrip = id
}

// Tick runs every few minutes: it notices when Ford has gone quiet for hours.
func (e *Engine) Tick() {
	e.mu.Lock()
	e.plugReminder()
	e.weeklyDigest()
	e.diskCheck()
	e.budget()
	e.backupCheck()
	if e.prev != nil {
		e.maintenance(e.prev.OdometerKm)
	}
	e.mu.Unlock()
	if e.Link == nil {
		return
	}
	linked, lastOK := e.Link()
	e.mu.Lock()
	defer e.mu.Unlock()
	if !linked || lastOK == 0 {
		return
	}
	silent := e.now().Sub(time.Unix(lastOK, 0))
	switch {
	case silent > noReadingsFor && !e.silentSent:
		e.silentSent = true
		e.send(push.Alert{Kind: push.FordTrouble, Title: "No readings from the car",
			Body: fmt.Sprintf("The last reading was %d hours ago. Open Status to see why.", int(silent.Hours())), URL: "#/status"})
	case silent < time.Hour:
		e.silentSent = false
	}
}

// Run checks for a silent connection every ten minutes until ctx ends.
func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Minute):
			e.Tick()
		}
	}
}
