// Package vhealth turns one flattened reading into the Vehicle health page: voltages, battery
// health, oil and fuel, warnings and codes. It shows only what the car reports, and only the
// parts that apply to its type (a battery car never gets oil life, even though Ford lists it).
package vhealth

import (
	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Row is one line on the page. Kind tells the page how to format Num (volts, pct, kwh, tempC...).
type Row struct {
	Label string   `json:"label"`
	Kind  string   `json:"kind"`
	Num   *float64 `json:"num,omitempty"`
	Text  string   `json:"text,omitempty"`
	State string   `json:"state,omitempty"` // ok | warn | bad | info
	Note  string   `json:"note,omitempty"`
	At    string   `json:"at,omitempty"` // when Ford last updated this value
}

// Series is a value over time, one point per day.
type Series struct {
	Name  string      `json:"name"`
	Label string      `json:"label"`
	Kind  string      `json:"kind"`
	Days  []store.Day `json:"days"`
}

type Section struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Intro  string   `json:"intro,omitempty"`
	Rows   []Row    `json:"rows"`
	Series []Series `json:"series,omitempty"`
}

type Report struct {
	Sections []Section `json:"sections"`
	Overall  string    `json:"overall"` // ok | warn | bad
}

// Trends are the daily values already read from the database, by series name.
type Trends map[string][]store.Day

// Build assembles the report for a car of the given type from its latest flattened reading.
func Build(f map[string]any, pt ford.Powertrain, trends Trends) Report {
	var secs []Section
	add := func(s Section) {
		if len(s.Rows) > 0 {
			secs = append(secs, s)
		}
	}
	add(attention(f))
	add(twelveVolt(f, trends))
	if pt.HasBattery() {
		add(traction(f, trends))
	}
	// An unknown type shows engine rows only when a fuel level proves there is one: battery cars list oil life too.
	if pt.HasEngine() && (pt != ford.Unknown || ford.Number(f, "fuelLevel") != nil) {
		add(engine(f, pt, trends))
	}
	add(tires(f))
	add(system(f))
	return Report{Sections: secs, Overall: overall(secs)}
}

func overall(secs []Section) string {
	worst := "ok"
	for _, s := range secs {
		for _, r := range s.Rows {
			switch r.State {
			case "bad":
				return "bad"
			case "warn":
				worst = "warn"
			}
		}
	}
	return worst
}

// num builds a numeric row from the first metric found; nil when the car does not report it.
func num(f map[string]any, label, kind string, names ...string) *Row {
	v := ford.Number(f, names...)
	if v == nil {
		return nil
	}
	return &Row{Label: label, Kind: kind, Num: v, At: ford.Updated(f, names[0])}
}

// text builds a text row; nil when the car does not report it.
func text(f map[string]any, label string, names ...string) *Row {
	v := ford.Text(f, names...)
	if v == "" {
		return nil
	}
	return &Row{Label: label, Kind: "text", Text: v, At: ford.Updated(f, names[0])}
}

// rows drops the metrics the car did not report.
func rows(rs ...*Row) []Row {
	out := []Row{}
	for _, r := range rs {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// series keeps a trend only when it spans at least two days.
func series(t Trends, name, label, kind string) []Series {
	if len(t[name]) < 2 {
		return nil
	}
	return []Series{{Name: name, Label: label, Kind: kind, Days: t[name]}}
}

// with sets the state and note of a row (nil stays nil).
func (r *Row) with(state, note string) *Row {
	if r != nil {
		r.State, r.Note = state, note
	}
	return r
}
