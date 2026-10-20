package cardata

import (
	"fmt"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// body is the Body and security tab: locks, doors, windows, tires, seat belts and the alarm.
func (s src) body() Tab {
	locks := group("Locks", "", s.listRows(ford.ListItems(s.f, "doorLockStatus"), "Whether each door is locked. Double locked means the extra deadbolt is on.", func(d map[string]any) string { return doorName(d) }, "doorLockStatus")...)
	doorItems := ford.ListItems(s.f, "doorStatus")
	doors := group("Doors", "", s.listRows(doorItems, "Closed is what you want. Ajar means a door is not fully shut. Ford reports both front doors as one entry, so it cannot say which of them is open.", doorNamer(doorItems), "doorStatus")...)
	hood := s.text("Hood", "Whether the hood is shut.", "hoodStatus")
	if doors != nil && hood != nil {
		doors.Rows = append(doors.Rows, *hood)
	}
	windows := group("Windows", "How open each window is. Closed means fully up.", s.windowRows()...)
	belts := group("Seat belts", "", s.listRows(ford.ListItems(s.f, "seatBeltStatus"), "Whether the belt of that seat is fastened.", func(d map[string]any) string {
		return words(fmt.Sprint(d["vehicleoccupantrole"])) + " seat belt"
	}, "seatBeltStatus")...)
	security := group("Alarm and remote start", "",
		s.text("Alarm", "Armed means the car's alarm is on and watching.", "alarmStatus"),
		s.text("Panic alarm", "Whether the panic alarm (the horn and lights) is sounding.", "panicAlarmStatus"),
		s.num("Remote start time left", "seconds", "After a remote start the engine or climate system shuts off if you do not get in. This is the time left. Zero means no remote start is running.", "remoteStartCountdownTimer"))
	return tab("body", "Body and security", "Doors, locks, windows, tires and the alarm.", locks, doors, windows, belts, s.tires(), security)
}

// listRows makes one row per item of a list metric such as doorStatus.
func (s src) listRows(items []map[string]any, help string, label func(map[string]any) string, raw string) []*Row {
	var out []*Row
	for i, it := range items {
		v := strings.TrimSpace(fmt.Sprint(it["value"]))
		if v == "" || v == "<nil>" {
			continue
		}
		r := &Row{Label: label(it), Kind: "text", Text: words(v), Raw: raw}
		if i == 0 {
			r.Help = help
		}
		out = append(out, r)
	}
	return out
}

func (s src) windowRows() []*Row {
	var out []*Row
	for _, w := range ford.ListItems(s.f, "windowStatus") {
		lo, hi := ford.Number(w, "value.doublerange.lowerbound"), ford.Number(w, "value.doublerange.upperbound")
		text := "Closed"
		if hi != nil && *hi > 0 {
			text = fmt.Sprintf("Open up to %.0f%%", *hi)
		} else if lo == nil && hi == nil {
			continue
		}
		pos := strings.ReplaceAll(strings.ToLower(fmt.Sprint(w["vehiclewindow"])), "unspecified_", "")
		side := strings.ToLower(fmt.Sprint(w["vehicleside"]))
		label := words(strings.TrimSpace(side + "_" + pos + "_window"))
		out = append(out, &Row{Label: label, Kind: "text", Text: text, Raw: "windowStatus"})
	}
	return out
}

func (s src) tires() *Group {
	items := ford.ListItems(s.f, "tirePressure")
	status := map[string]string{}
	for _, it := range ford.ListItems(s.f, "tirePressureStatus") {
		status[fmt.Sprint(it["vehiclewheel"])] = words(fmt.Sprint(it["value"]))
	}
	var rows []*Row
	var front, rear *float64
	for _, it := range items {
		wheel := fmt.Sprint(it["vehiclewheel"])
		v := ford.Number(map[string]any{"v": it["value"]}, "v")
		if v == nil {
			continue
		}
		rows = append(rows, &Row{Label: words(wheel), Kind: "kpa", Num: v, Text: status[wheel], Raw: "tirePressure"})
		if p := ford.Number(map[string]any{"p": it["wheelplacardfront"]}, "p"); p != nil {
			front = p
		}
		if p := ford.Number(map[string]any{"p": it["wheelplacardrear"]}, "p"); p != nil {
			rear = p
		}
	}
	if len(rows) > 0 {
		rows[0].Help = "Pressure in each tire, with the car's own verdict next to it."
	}
	if front != nil {
		rows = append(rows, &Row{Label: "Recommended (front)", Kind: "kpa", Num: front, Help: "The pressure printed on the sticker in the driver's door frame.", Raw: "wheelPlacardFront"})
	}
	if rear != nil {
		rows = append(rows, &Row{Label: "Recommended (rear)", Kind: "kpa", Num: rear, Raw: "wheelPlacardRear"})
	}
	rows = append(rows, s.text("Tire monitor", "Whether the car's tire pressure warning system is working.", "tirePressureSystemStatus"))
	return group("Tires", "", rows...)
}
