// Package mileage builds a business mileage log from work trips: one row per day with the origin
// city, the cities where the car was parked (each once, in order), a description and the miles.
package mileage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/geocode"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Places turns a location into a city. Cached answers are free; a lookup may go out to the
// internet, so a run only makes a limited number of them.
type Places interface {
	Cached(lat, lon float64) (string, bool)
	Lookup(ctx context.Context, lat, lon float64) (string, error)
}

type Options struct {
	Description string // the Description column; default "Customer Visits"
	MaxLookups  int    // new places to look up in one run; default 40
	Default     *time.Location
}

type Row struct {
	Date         string `json:"date"`
	Origin       string `json:"origin"`
	Destinations string `json:"destinations"`
	Description  string `json:"description"`
	Miles        int    `json:"miles"`
	Trips        int    `json:"trips"`
}

type Report struct {
	Rows       []Row    `json:"rows"`
	TotalMiles int      `json:"totalMiles"`
	Notes      []string `json:"notes"`
	Pending    int      `json:"pending"` // places not looked up yet; run again to finish
}

const kmToMiles = 0.621371

type resolver struct {
	pl       Places
	max      int
	lookups  int
	memo     map[string]string
	pending  map[string]bool
	firstErr error
}

// city names a place: from memory, the cache, or a fresh lookup while the run's allowance lasts.
func (r *resolver) city(ctx context.Context, lat, lon *float64) (string, bool) {
	if lat == nil || lon == nil {
		return "", false
	}
	key := geocode.Key(*lat, *lon)
	if c, ok := r.memo[key]; ok {
		return c, true
	}
	if c, ok := r.pl.Cached(*lat, *lon); ok {
		r.memo[key] = c
		return c, true
	}
	if r.lookups >= r.max || ctx.Err() != nil || r.firstErr != nil {
		r.pending[key] = true
		return "", false
	}
	r.lookups++
	c, err := r.pl.Lookup(ctx, *lat, *lon)
	if err != nil {
		r.firstErr = err // the service is refusing or unreachable: stop asking in this run
		r.pending[key] = true
		return "", false
	}
	r.memo[key] = c
	return c, true
}

// Build makes the log for the given trips (any order). Open trips are left out.
func Build(ctx context.Context, trips []*store.Trip, pl Places, opt Options) Report {
	if opt.Description == "" {
		opt.Description = "Customer Visits"
	}
	if opt.MaxLookups <= 0 {
		opt.MaxLookups = 40
	}
	if opt.Default == nil {
		opt.Default = time.UTC
	}
	sort.Slice(trips, func(i, j int) bool { return trips[i].StartTS < trips[j].StartTS })
	days := map[string][]*store.Trip{}
	var order []string
	for _, t := range trips {
		if t.Open || t.EndTS == nil {
			continue
		}
		d := time.Unix(t.StartTS, 0).In(zone(t, opt.Default)).Format("2006-01-02")
		if _, seen := days[d]; !seen {
			order = append(order, d)
		}
		days[d] = append(days[d], t)
	}
	r := &resolver{pl: pl, max: opt.MaxLookups, memo: map[string]string{}, pending: map[string]bool{}}
	rep := Report{Rows: []Row{}, Notes: []string{}}
	for _, d := range order {
		row := dayRow(ctx, r, d, days[d], opt.Description, &rep)
		rep.Rows = append(rep.Rows, row)
		rep.TotalMiles += row.Miles
	}
	rep.Pending = len(r.pending)
	if rep.Pending > 0 {
		why := "the limit of new lookups per run was reached"
		if r.firstErr != nil {
			why = "the place lookup service did not answer (" + r.firstErr.Error() + ")"
		}
		rep.Notes = append(rep.Notes, fmt.Sprintf("%d place(s) could not be named yet because %s. Run the report again to finish; answers already found are kept.", rep.Pending, why))
	}
	return rep
}

func zone(t *store.Trip, def *time.Location) *time.Location {
	if t.TZ != "" {
		if l, err := time.LoadLocation(t.TZ); err == nil {
			return l
		}
	}
	return def
}

func dayRow(ctx context.Context, r *resolver, date string, ts []*store.Trip, desc string, rep *Report) Row {
	var km float64
	for _, t := range ts {
		km += t.DistanceKm
	}
	row := Row{Date: date, Description: desc, Trips: len(ts), Miles: int(math.Round(km * kmToMiles))}
	if c, ok := r.city(ctx, ts[0].StartLat, ts[0].StartLon); ok {
		row.Origin = c
	}
	seen := map[string]bool{strings.ToLower(row.Origin): row.Origin != ""} // the origin is home: not listed as a destination
	var dest []string
	missing := 0
	for _, t := range ts {
		c, ok := r.city(ctx, t.EndLat, t.EndLon)
		if !ok {
			if t.EndLat == nil || t.EndLon == nil {
				missing++
			}
			continue
		}
		if c = strings.TrimSpace(c); c == "" || seen[strings.ToLower(c)] {
			continue
		}
		seen[strings.ToLower(c)] = true
		dest = append(dest, c)
	}
	row.Destinations = strings.Join(dest, ", ")
	if missing > 0 {
		rep.Notes = append(rep.Notes, fmt.Sprintf("%s: %d trip(s) have no end location, so their city is missing.", date, missing))
	}
	return row
}

// TSV is the log as tab-separated text that pastes into a spreadsheet cell by cell. With the header
// it also carries the column titles and the Total Mileage Recorded line.
func (rep Report) TSV(withHeader bool) string {
	var b strings.Builder
	if withHeader {
		b.WriteString("Date\tOrigin City\tDestination City\tDescription\tTotal Miles Driven\n")
	}
	for _, r := range rep.Rows {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%d\n", r.Date, r.Origin, r.Destinations, r.Description, r.Miles)
	}
	if withHeader {
		fmt.Fprintf(&b, "\nTotal Mileage Recorded:\t\t\t\t%d\n", rep.TotalMiles)
	}
	return b.String()
}
