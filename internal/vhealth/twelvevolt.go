package vhealth

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// rechargedRecently says the daily readings of the last three days include one from a running car (a charging system reads 13 V or
// more), so a low reading from when it was parked is not the whole story.
func rechargedRecently(days []store.Day) bool {
	for i := max(0, len(days)-3); i < len(days); i++ {
		if days[i].Max >= 13.2 {
			return true
		}
	}
	return false
}

// twelveVolt is the small battery that runs the electronics. Its charge is the car's own estimate, not a
// test, and a parked car often reads lowish until it next wakes and recharges it, so the lines for a
// warning are deliberately low. A shop test the owner has recorded as good silences the warnings.
func twelveVolt(f map[string]any, t Trends, opts Options) Section {
	recharged := rechargedRecently(t["battery12v"])
	v := num(f, "Voltage", "volts", "batteryVoltage")
	if v != nil {
		switch {
		case *v.Num < 11.6:
			v.with("bad", "Very low. The car may struggle to wake or start. Have the 12 V battery tested.")
		case *v.Num < 11.9:
			v.with("warn", "Low even for a parked car. Worth watching, and worth a test if it stays this low.")
		case *v.Num < 12.3:
			v.with("ok", "A little below a full battery, which is normal for a car that has been parked. It recovers when the car wakes and recharges it.")
		default:
			v.with("ok", "")
		}
	}
	c := num(f, "Charge", "pct", "batteryStateOfCharge")
	if c != nil {
		switch {
		case *c.Num < 30 && recharged:
			c.with("warn", "Low charge reading, but the car has charged this battery since (it read above 13 V while running in the last few days). A parked car reads low until it wakes, so look again after the next drive.")
		case *c.Num < 30:
			c.with("bad", "Low charge. Have the 12 V battery tested.")
		case *c.Num < 50:
			c.with("warn", "Below half. Worth watching if it stays low across several days.")
		case *c.Num < 70:
			c.with("ok", "Middling, which is common for a parked car. The car recharges it when it wakes.")
		default:
			c.with("ok", "")
		}
	}
	load := text(f, "Load", "batteryLoadStatus")
	if load != nil {
		load.State = "ok"
		if !ford.LoadNormal(load.Text) {
			load.State = "warn"
		}
		load.wordy()
	}
	if opts.Tested12VUntil.After(time.Now()) { // a proper test beats the car's estimate
		for _, r := range []*Row{v, c, load} {
			if r != nil && (r.State == "warn" || r.State == "bad") {
				r.State, r.Note = "ok", "You had this battery tested and it was fine, so reminders are paused until "+opts.Tested12VUntil.Format("Jan 2")+"."
			}
		}
	}
	return Section{ID: "12v", Title: "12 V battery",
		Intro: "The small battery that runs the lights, screens and computers and wakes the car. The percentage is the car's own estimate, not a test, and a parked car often reads lower until it next wakes. A shop test is the better measure of its health.",
		Rows:  rows(v, c, load), Series: series(t, "battery12v", "Voltage, daily average", "volts")}
}
