package calendar

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Occurrence is one appointment on one day.
type Occurrence struct {
	UID      string    `json:"uid"`
	Title    string    `json:"title"`
	Location string    `json:"location"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"allDay"`
}

const maxSteps = 5000 // a rule is never followed further than this, so a broken one cannot run for ever

// Between lists the appointments that overlap [from, to), sorted by start. Repeating appointments are expanded for the rules
// calendars write: daily, weekly (with days and every other week), monthly (by date or "the second Tuesday") and yearly, with a
// count or an end date, dates taken out and single occurrences moved or cancelled.
func Between(events []Event, from, to time.Time) []Occurrence {
	moved := map[string]bool{}
	var out []Occurrence
	in := func(s, e time.Time) bool { return s.Before(to) && e.After(from) }
	for _, ev := range events {
		if ev.RecurrenceID != nil {
			moved[ev.UID+"|"+strconv.FormatInt(ev.RecurrenceID.Unix(), 10)] = true
			if !ev.Cancelled && in(ev.Start, ev.End) {
				out = append(out, occ(ev, ev.Start, ev.End))
			}
		}
	}
	for _, ev := range events {
		if ev.RecurrenceID != nil || ev.Cancelled {
			continue
		}
		if ev.Rule == "" {
			if in(ev.Start, ev.End) {
				out = append(out, occ(ev, ev.Start, ev.End))
			}
			continue
		}
		length := ev.End.Sub(ev.Start)
		expand(ev, to, func(s time.Time) {
			if moved[ev.UID+"|"+strconv.FormatInt(s.Unix(), 10)] || excepted(ev, s) || !in(s, s.Add(length)) {
				return
			}
			out = append(out, occ(ev, s, s.Add(length)))
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func occ(ev Event, s, e time.Time) Occurrence {
	return Occurrence{UID: ev.UID, Title: ev.Title, Location: ev.Location, Start: s, End: e, AllDay: ev.AllDay}
}

func excepted(ev Event, s time.Time) bool {
	for _, x := range ev.Except {
		if x.Equal(s) || (ev.AllDay && x.Format("20060102") == s.Format("20060102")) {
			return true
		}
	}
	return false
}

var weekdays = map[string]time.Weekday{"SU": time.Sunday, "MO": time.Monday, "TU": time.Tuesday, "WE": time.Wednesday, "TH": time.Thursday, "FR": time.Friday, "SA": time.Saturday}

// expand calls each with every start the rule makes, in order, until the rule ends or a start is at or past to.
func expand(ev Event, to time.Time, each func(time.Time)) {
	r := map[string]string{}
	for _, p := range strings.Split(ev.Rule, ";") {
		if k, v, ok := strings.Cut(p, "="); ok {
			r[strings.ToUpper(k)] = strings.ToUpper(v)
		}
	}
	step, _ := strconv.Atoi(r["INTERVAL"])
	if step < 1 {
		step = 1
	}
	count, _ := strconv.Atoi(r["COUNT"])
	until := time.Time{}
	if u := r["UNTIL"]; u != "" {
		until, _ = parseTime(u, nil, ev.Start.Location())
		if len(u) == 8 {
			until = until.Add(24*time.Hour - time.Second) // a date: the whole of that day
		}
	}
	made := 0
	emit := func(s time.Time) bool { // false once the rule is over
		if s.Before(ev.Start) {
			return true
		}
		if (count > 0 && made >= count) || (!until.IsZero() && s.After(until)) || !s.Before(to) {
			return false
		}
		made++
		each(s)
		return true
	}
	s := ev.Start
	h, m, sec := s.Clock()
	at := func(y int, mo time.Month, d int) time.Time { return time.Date(y, mo, d, h, m, sec, 0, s.Location()) }
	switch r["FREQ"] {
	case "DAILY":
		for i := 0; i < maxSteps && emit(at(s.Year(), s.Month(), s.Day()+i*step)); i++ {
		}
	case "WEEKLY":
		days := byDays(r["BYDAY"], s.Weekday())
		monday := at(s.Year(), s.Month(), s.Day()-(int(s.Weekday())+6)%7)
		for w := 0; w < maxSteps; w += step {
			for _, d := range days {
				if !emit(monday.AddDate(0, 0, w*7+(int(d)+6)%7)) {
					return
				}
			}
		}
	case "MONTHLY":
		for i := 0; i < maxSteps; i += step {
			first := at(s.Year(), s.Month()+time.Month(i), 1)
			day := nthWeekday(first, r["BYDAY"])
			if day.IsZero() && r["BYDAY"] != "" {
				continue // no fifth Monday this month
			}
			if day.IsZero() {
				day = at(first.Year(), first.Month(), s.Day())
				if day.Month() != first.Month() { // the 31st in a shorter month: skipped, as calendars do
					continue
				}
			}
			if !emit(day) {
				return
			}
		}
	case "YEARLY":
		for i := 0; i < maxSteps && emit(at(s.Year()+i*step, s.Month(), s.Day())); i++ {
		}
	}
}

// byDays reads BYDAY for a weekly rule (MO,WE,FR), in week order; the start's own day when it has none.
func byDays(v string, own time.Weekday) []time.Weekday {
	var out []time.Weekday
	for _, p := range strings.Split(v, ",") {
		if d, ok := weekdays[strings.TrimLeft(p, "+-0123456789")]; ok {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		out = []time.Weekday{own}
	}
	sort.Slice(out, func(i, j int) bool { return (int(out[i])+6)%7 < (int(out[j])+6)%7 })
	return out
}

// nthWeekday is "2TU" (the second Tuesday) or "-1FR" (the last Friday) of first's month, or zero when v is not that form.
func nthWeekday(first time.Time, v string) time.Time {
	if v == "" || strings.Contains(v, ",") {
		return time.Time{}
	}
	d, ok := weekdays[v[len(v)-2:]]
	n, err := strconv.Atoi(v[:len(v)-2])
	if !ok || err != nil || n == 0 {
		return time.Time{}
	}
	if n > 0 {
		t := first.AddDate(0, 0, (int(d)-int(first.Weekday())+7)%7+(n-1)*7)
		if t.Month() != first.Month() {
			return time.Time{}
		}
		return t
	}
	last := first.AddDate(0, 1, -1)
	return last.AddDate(0, 0, -((int(last.Weekday())-int(d)+7)%7)+(n+1)*7)
}
