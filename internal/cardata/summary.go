package cardata

import (
	"fmt"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// summary is the first tab: the few things worth a glance, and how fresh the data is.
func (s src) summary() Tab {
	st := ford.ParseFlat(s.f, s.asOf)
	doing := "Parked"
	switch {
	case st.Charging():
		doing = "Charging"
	case st.Driving():
		doing = "Driving or switched on"
	case strings.EqualFold(st.Plug, "connected"):
		doing = "Plugged in, not charging"
	}
	now := group("Right now", "",
		say("State", doing, "What the car is doing in this reading."),
		s.battery0(), s.num("Range", "km", "How far the car expects to go on what is left.", "xevBatteryRange"),
		s.num("Energy left", "kwh", "Usable energy left in the battery, like gallons in a fuel tank. A kilowatt-hour (kWh) is the unit electricity is sold in.", "xevBatteryEnergyRemaining"),
		s.fuel0(), s.num("Odometer", "km", "The total distance the car has driven.", "odometer"),
		s.num("Outside temperature", "tempC", "The air temperature outside the car.", "outsideTemperature", "ambientTemp"),
		s.doorSummary())
	fresh := group("How fresh is this data?", "Ford does not send every value every time. A parked car reports some of them only when it wakes up (after a drive, a charge or a lock). Each value says when it was last updated.",
		s.freshness(), s.plain("Last update from Ford", "When Ford last refreshed this car's data.", "telemetry.updateTime"))
	return tab("summary", "Summary", "The essentials in plain words.", now, fresh)
}

func (s src) battery0() *Row {
	if !s.pt.HasBattery() {
		return nil
	}
	return s.num("Charge", "pct", "The battery percentage shown on the car's display.", "xevBatteryStateOfCharge")
}

func (s src) fuel0() *Row {
	if s.pt == ford.EV || (s.pt == ford.Unknown && ford.Number(s.f, "fuelLevel") == nil) {
		return nil
	}
	return s.num("Fuel", "pct", "How full the fuel tank is.", "fuelLevel")
}

// doorSummary says in one line whether the car is shut and locked.
func (s src) doorSummary() *Row {
	locks, doors := ford.ListItems(s.f, "doorLockStatus"), ford.ListItems(s.f, "doorStatus")
	if len(locks) == 0 && len(doors) == 0 {
		return nil
	}
	var open []string
	for _, d := range doors {
		if v := strings.ToUpper(fmt.Sprint(d["value"])); v != "CLOSED" && v != "UNKNOWN" && v != "" {
			open = append(open, doorName(d))
		}
	}
	if hood := strings.ToUpper(ford.Text(s.f, "hoodStatus")); hood != "" && hood != "CLOSED" {
		open = append(open, "Hood")
	}
	unlocked := false
	for _, l := range locks {
		if v := strings.ToUpper(fmt.Sprint(l["value"])); v == "UNLOCKED" {
			unlocked = true
		}
	}
	text := "All closed and locked"
	switch {
	case len(open) > 0:
		text = "Open: " + strings.Join(open, ", ")
	case unlocked:
		text = "Closed, but unlocked"
	case len(locks) == 0:
		text = "All closed"
	}
	return say("Doors and locks", text, "Whether every door, the tailgate and the hood are shut, and the car is locked.")
}

// freshness counts how many values are recent compared with the moment of the reading.
func (s src) freshness() *Row {
	total, recent := 0, 0
	var oldest time.Time
	for k, v := range s.f {
		if !strings.HasPrefix(k, "telemetry.metrics.") || !strings.HasSuffix(k, ".updateTime") {
			continue
		}
		str, _ := v.(string)
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			continue
		}
		total++
		if s.asOf.Sub(t) <= 2*time.Hour {
			recent++
		}
		if oldest.IsZero() || t.Before(oldest) {
			oldest = t
		}
	}
	if total == 0 {
		return nil
	}
	text := fmt.Sprintf("%d of %d values were updated within 2 hours", recent, total)
	if !oldest.IsZero() && s.asOf.Sub(oldest) > 2*time.Hour {
		text += "; the oldest is from " + oldest.Local().Format("Jan 2")
	}
	return say("Values updated recently", text, "A high number means the car has been awake and reporting. A low number is normal for a car that has been parked a while.")
}

func doorName(d map[string]any) string {
	side := strings.ToUpper(fmt.Sprint(d["vehicleside"]))
	door := strings.ReplaceAll(strings.ToLower(fmt.Sprint(d["vehicledoor"])), "unspecified_", "")
	door = strings.ReplaceAll(door, "_", " ")
	if side != "" && side != "UNKNOWN" && side != "<NIL>" {
		door = strings.ToLower(side) + " " + door
	}
	if door == "all doors" {
		return "All doors"
	}
	return words(strings.ReplaceAll(door, " ", "_"))
}
