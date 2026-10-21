// Package alerts decides when something is worth a notification, by comparing each new reading
// of the car with the one before it. It sends each alert once, not on every reading.
package alerts

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Notifier delivers an alert to the devices that want it.
type Notifier interface {
	Notify(ctx context.Context, a push.Alert)
}

// Link says how the connection to Ford is doing, for the "no readings" alert.
type Link func() (linked bool, lastOK int64)

const (
	unlockedAfter = 10 * time.Minute
	lowVoltage    = 11.8 // volts: same line the Health page calls "act"
	lowChargePct  = 50.0 // 12 V battery charge percent
	noReadingsFor = 4 * time.Hour
	kv12V         = "alert_12v_at"
)

type Engine struct {
	S     *store.Store
	N     Notifier
	Link  Link
	Async bool             // send without waiting (the real server); tests send in line
	Now   func() time.Time // for tests

	// The evening plug-in reminder: the time of day ("" is off), the battery level, and the local time zone.
	ReminderTime    func() string
	ReminderBelow   func() float64
	DiskSpace       func() (free, total uint64, ok bool) // room on the data disk; nil skips the check
	LowBatteryLevel func() float64                       // the battery reserve, warned about on devices that turned the alert on; 0 or nil is off
	ChargeBudget    func() float64                       // monthly charging budget; 0 or nil is none
	MonthCost       func() (float64, bool)               // what this month's charging has cost, when it can be told
	BumpLevel       func() float64                       // the motion level that counts as a bump while parked; 0 or nil is off
	Loc             func() *time.Location

	mu           sync.Mutex
	prev         *ford.State
	started      bool // the first reading only sets the baseline: nothing is "new" after a restart
	openTrip     int64
	tireProblem  bool
	unlockedFrom time.Time
	unlockedSent bool
	silentSent   bool
	inside       map[int64]bool // which named places the car is inside
	lastBump     time.Time
	lastDrive    time.Time   // the last reading with the ignition on or the car moving
	restVec      *[3]float64 // the previous parked reading of the motion sensors
	lowSent      bool
	openFrom     time.Time
	openSent     bool
	openKey      string // what the last Left open alert was about, so the same old report never alerts twice
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) send(a push.Alert) {
	if e.Async {
		go e.N.Notify(context.Background(), a)
		return
	}
	e.N.Notify(context.Background(), a)
}

// Handle looks at a reading that has been saved and run through the trip detector.
func (e *Engine) Handle(st *ford.State) {
	e.mu.Lock()
	defer e.mu.Unlock()
	first := !e.started
	e.started = true
	if !first && e.prev != nil {
		e.plugAndCharge(e.prev, st)
	}
	e.tires(st, first)
	e.battery12V(st)
	e.carFaults(st)
	e.maintenance(st.OdometerKm)
	e.namedPlaces(st, first)
	e.parkedWatch(st, first)
	e.lowBattery(st, first)
	e.unlocked(st)
	e.leftOpen(st)
	e.trip(first)
	e.prev = st
}

func (e *Engine) plugAndCharge(prev, st *ford.State) {
	was, wasKnown := plugged(prev)
	now, nowKnown := plugged(st)
	switch {
	case wasKnown && nowKnown && !was && now:
		e.send(push.Alert{Kind: push.PlugIn, Title: "Plugged in", Body: "Charging cable connected." + charge(st), URL: "#/charging"})
	case wasKnown && nowKnown && was && !now:
		e.send(push.Alert{Kind: push.PlugOut, Title: "Unplugged", Body: "Charging cable removed." + charge(st), URL: "#/charging"})
	}
	if charging(prev) && !charging(st) {
		e.chargeSpeed()
	}
	if charging(prev) && !charging(st) && nowKnown && now { // stopped while still plugged in
		soc := "?"
		if st.SoCPct != nil {
			soc = fmt.Sprintf("%.0f%%", *st.SoCPct)
		}
		if why := chargeFault(st); why != "" {
			e.S.LogDecision("charge_stop", fmt.Sprintf("charging stopped at %s, state %s: sent 'interrupted' (%s)", soc, st.ChargeState, strings.TrimSpace(why)))
			e.send(push.Alert{Kind: push.ChargeInterrupted, Title: "Charging interrupted", Body: why + strings.TrimRight(charge(st), " "), URL: "#/charging"})
		} else if e.stoppedEarly(st) {
			e.S.LogDecision("charge_stop", fmt.Sprintf("charging stopped at %s, state %s: sent 'stopped early' (lower than your sessions usually end)", soc, st.ChargeState))
			e.send(push.Alert{Kind: push.ChargeInterrupted, Title: "Charging stopped early", Body: fmt.Sprintf("Charging stopped at %.0f%% while still plugged in, lower than it usually stops. Check the cable and the charger.", *st.SoCPct), URL: "#/charging"})
		} else {
			e.S.LogDecision("charge_stop", fmt.Sprintf("charging stopped at %s, state %s: sent 'complete'", soc, st.ChargeState))
			e.send(push.Alert{Kind: push.ChargeComplete, Title: "Charging complete", Body: strings.TrimSpace(charge(st)), URL: "#/charging"})
		}
	}
}

func (e *Engine) tires(st *ford.State, first bool) {
	var low []string
	for _, it := range ford.ListItems(st.Extra, "tirePressureStatus") {
		v := strings.ToUpper(fmt.Sprint(it["value"]))
		if v == "" || v == "NORMAL" || v == "UNKNOWN" || v == "UNSUPPORTED" || v == "INVALID" || v == "NORMAL_OPERATION" {
			continue
		}
		low = append(low, wheel(fmt.Sprint(it["vehiclewheel"])))
	}
	problem := len(low) > 0
	if problem && !e.tireProblem && !first {
		e.send(push.Alert{Kind: push.LowTire, Title: "Low tire pressure", Body: strings.Join(low, ", ") + " reported by the car.", URL: "#/health"})
	}
	e.tireProblem = problem
}

// battery12V alerts at most once a day, and the time is kept so a restart does not repeat it.
func (e *Engine) battery12V(st *ford.State) {
	if n, err := strconv.ParseInt(e.S.GetKV(store.KV12VTested), 10, 64); err == nil && e.now().Before(time.Unix(n, 0)) {
		return // the owner had the battery tested and it was fine
	}
	soc := ford.Number(st.Extra, "batteryStateOfCharge")
	low := (st.Battery12V != nil && *st.Battery12V < lowVoltage) || (soc != nil && *soc < lowChargePct)
	if !low {
		return
	}
	if at, err := strconv.ParseInt(e.S.GetKV(kv12V), 10, 64); err == nil && e.now().Sub(time.Unix(at, 0)) < 24*time.Hour {
		return
	}
	_ = e.S.SetKV(kv12V, strconv.FormatInt(e.now().Unix(), 10))
	body := "The 12 V battery is low"
	if st.Battery12V != nil {
		body += fmt.Sprintf(": %.1f V", *st.Battery12V)
		if soc != nil {
			body += fmt.Sprintf(" (%.0f%%)", *soc)
		}
	}
	e.send(push.Alert{Kind: push.Low12V, Title: "Low 12 V battery", Body: body + ". The car may not wake if it drops further.", URL: "#/health"})
}

func (e *Engine) unlocked(st *ford.State) {
	if st.Locked == nil || *st.Locked || st.Driving() { // ignition on means someone is in the car, even while it charges
		e.unlockedFrom, e.unlockedSent = time.Time{}, false
		return
	}
	if e.unlockedFrom.IsZero() {
		e.unlockedFrom = st.TS
	}
	if !e.unlockedSent && st.TS.Sub(e.unlockedFrom) >= unlockedAfter {
		e.unlockedSent = true
		e.send(push.Alert{Kind: push.Unlocked, Title: "Car left unlocked", Body: "Parked and unlocked for over 10 minutes.", URL: "#/"})
	}
}
