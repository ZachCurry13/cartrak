// Package cardata turns one flattened reading from Ford into the Car data page: friendly names, units,
// plain-English explanations and how old each value is, grouped into tabs. It shows only what the car
// reports and never includes the VIN or a location.
package cardata

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// Row is one value. Kind tells the page how to format Num (volts, pct, kwh, km, tempC...); Help says in
// plain words what it means; Raw is Ford's own name for it, shown to the curious.
type Row struct {
	Label string   `json:"label"`
	Kind  string   `json:"kind"`
	Num   *float64 `json:"num,omitempty"`
	Text  string   `json:"text,omitempty"`
	Help  string   `json:"help,omitempty"`
	Raw   string   `json:"raw,omitempty"`
	At    string   `json:"at,omitempty"` // when Ford last updated this value
}

type Group struct {
	Title string `json:"title"`
	Intro string `json:"intro,omitempty"`
	Rows  []Row  `json:"rows"`
}

type Tab struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Intro  string  `json:"intro,omitempty"`
	Groups []Group `json:"groups"`
}

type Report struct {
	Tabs   []Tab `json:"tabs"`
	Fields int   `json:"fields"` // how many separate values Ford sent in this reading
}

// src is the reading being described.
type src struct {
	f    map[string]any
	asOf time.Time
	pt   ford.Powertrain
}

// Build describes the reading f, taken at asOf, for a car of the given type.
func Build(f map[string]any, asOf time.Time, pt ford.Powertrain) Report {
	s := src{f: f, asOf: asOf, pt: pt}
	tabs := []Tab{s.summary(), s.battery(), s.driving(), s.body(), s.settings(), s.car()}
	out := make([]Tab, 0, len(tabs))
	for _, t := range tabs {
		if len(t.Groups) > 0 {
			out = append(out, t)
		}
	}
	return Report{Tabs: out, Fields: len(f)}
}

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// words turns NORMAL_OPERATION or dieselParticulateFilter into "Normal operation" / "Diesel particulate filter".
func words(s string) string {
	t := strings.ToLower(strings.ReplaceAll(camel.ReplaceAllString(s, "$1 $2"), "_", " "))
	if t == "" {
		return ""
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

// num builds a numeric row from the first metric found; nil when the car does not report it.
func (s src) num(label, kind, help string, names ...string) *Row {
	v := ford.Number(s.f, names...)
	if v == nil {
		return nil
	}
	return &Row{Label: label, Kind: kind, Num: v, Help: help, Raw: names[0], At: ford.Updated(s.f, names[0])}
}

// text builds a row from an enum or word value, made readable (NORMAL_OPERATION becomes "Normal operation").
func (s src) text(label, help string, names ...string) *Row {
	v := ford.Text(s.f, names...)
	if v == "" {
		return nil
	}
	return &Row{Label: label, Kind: "text", Text: words(v), Help: help, Raw: names[0], At: ford.Updated(s.f, names[0])}
}

// plain is a text row whose value is shown exactly as sent (a model name, a time).
func (s src) plain(label, help string, names ...string) *Row {
	v := ford.Text(s.f, names...)
	if v == "" {
		return nil
	}
	return &Row{Label: label, Kind: "text", Text: v, Help: help, Raw: names[0], At: ford.Updated(s.f, names[0])}
}

func say(label, text, help string) *Row {
	return &Row{Label: label, Kind: "text", Text: text, Help: help}
}

// group keeps only the rows that exist; a group with none is dropped.
func group(title, intro string, rs ...*Row) *Group {
	g := &Group{Title: title, Intro: intro, Rows: []Row{}}
	for _, r := range rs {
		if r != nil {
			g.Rows = append(g.Rows, *r)
		}
	}
	if len(g.Rows) == 0 {
		return nil
	}
	return g
}

func tab(id, title, intro string, gs ...*Group) Tab {
	t := Tab{ID: id, Title: title, Intro: intro, Groups: []Group{}}
	for _, g := range gs {
		if g != nil {
			t.Groups = append(t.Groups, *g)
		}
	}
	return t
}

// clock12 turns "07:30" into "7:30 AM".
func clock12(hhmm string) string {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return hhmm
	}
	return t.Format("3:04 PM")
}

func dayName(d string) string { return words(d) }

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
