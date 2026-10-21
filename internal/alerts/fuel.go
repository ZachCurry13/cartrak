package alerts

import (
	"fmt"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/fuel"
	"github.com/ZachCurry13/cartrak/internal/push"
)

const (
	lowFuelPct   = 15.0 // the tank at or below this is low
	lowFuelRearm = 25.0 // and the alert may come again once it has climbed past this
	fillWindow   = 30 * 60
)

// fuelAlerts watches the tank of a gas or hybrid car: a jump up is a fill-up (kept, to be completed with the gallons and the price
// per gallon, and asked about once), and a low tank is told once until it has been filled.
func (e *Engine) fuelAlerts(st *ford.State, first bool) {
	if st.FuelPct == nil {
		return
	}
	if !first && e.prev != nil && (st.SpeedKph == nil || *st.SpeedKph < 3) && fuel.Filled(e.prev.FuelPct, st.FuelPct) {
		e.filledUp(*e.prev.FuelPct, st)
	}
	switch {
	case *st.FuelPct >= lowFuelRearm:
		e.lowFuelSent = false
	case *st.FuelPct <= lowFuelPct && !e.lowFuelSent:
		e.lowFuelSent = true
		if !first {
			e.send(push.Alert{Kind: push.LowFuel, Title: "Low fuel", Body: fmt.Sprintf("The tank is at %.0f%%.", *st.FuelPct), URL: "#/fuel"})
		}
	}
}

// filledUp keeps a fill-up. Fueling takes minutes and the car reports every few, so a second jump soon after the first is the
// same fill-up reaching a higher level, not a new one.
func (e *Engine) filledUp(before float64, st *ford.State) {
	now := st.TS.Unix()
	if last, err := e.S.LastRefuel(); err == nil && last != nil && now-last.TS <= fillWindow && last.Status != "dismissed" && last.Source == "auto" {
		last.EndPct = *st.FuelPct
		_ = e.S.SaveRefuel(last)
		return
	}
	f := &fuel.Fill{TS: now, StartPct: before, EndPct: *st.FuelPct, OdoKm: st.OdometerKm, Status: "pending", Source: "auto"}
	if err := e.S.SaveRefuel(f); err != nil {
		return
	}
	e.S.LogDecision("fill_up", fmt.Sprintf("the tank went from %.0f%% to %.0f%%: kept as a fill-up to complete", before, *st.FuelPct))
	e.send(push.Alert{Kind: push.FillUp, Title: "Filled up?",
		Body: fmt.Sprintf("The tank went from %.0f%% to %.0f%%. Tap to enter the gallons and the price per gallon.", before, *st.FuelPct), URL: "#/fuel"})
}
