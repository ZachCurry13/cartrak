package server

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/fieldwatch"
)

// Everything the car reported over a stretch of time, as spreadsheets. The Car data page shows the latest reading; this is the
// history behind it: every field, and every moment any of them changed. Admin only (it is the raw data).

const fieldsReadme = `CarTrak car fields
==================
latest-reading.csv   every field of the newest reading, as the car sent it (field, value)
unchanged.csv        the fields that never changed in the stretch (field, value)
changes.csv          one row each time something changed: the time (UTC and local), then a column for every field that changed.
                     A reading where nothing changed is not repeated, so a parked car is a few rows and a drive is many.
Left out of changes.csv and unchanged.csv: the "updateTime" and "oemCorrelationId" of each metric (the car's own bookkeeping; the
newest ones are in latest-reading.csv).
Positions, the VIN and vehicle ids are left out of all three unless you asked for them. Times ending in Z are UTC.
`

func fieldValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case string:
		return t
	}
	return fmt.Sprint(v)
}

// bookkeeping says whether a field is the car's own freshness stamp, which changes with every report and says nothing new.
func bookkeeping(k string) bool {
	return strings.HasSuffix(k, ".updateTime") || strings.HasSuffix(k, ".oemCorrelationId")
}

func stop(ctx context.Context) error { return ctx.Err() }

// exportFields: GET /api/export/fields.zip?days=7&private=1
func (s *Server) exportFields(w http.ResponseWriter, r *http.Request) {
	days := intQuery(r, "days", 7, 1, 90)
	withPrivate := r.URL.Query().Get("private") == "1"
	loc := s.defaultLoc()
	to := time.Now().Unix()
	from := to - int64(days)*86400
	keep := func(k string) bool { return withPrivate || !fieldwatch.Private(k) }

	// First pass: which fields changed, and the newest reading.
	var first map[string]string
	seen := map[string]int{}
	changed := map[string]bool{}
	var latest map[string]any
	rows := 0
	err := s.S.EachExtra(from, to, func(ts int64, extra []byte) error {
		f := map[string]any{}
		if json.Unmarshal(extra, &f) != nil {
			return nil
		}
		rows++
		cur := map[string]string{}
		for k, v := range f {
			if keep(k) {
				cur[k] = fieldValue(v)
				seen[k]++
			}
		}
		if first == nil {
			first = cur
		} else {
			for k, v := range cur {
				if old, had := first[k]; !had || old != v {
					changed[k] = true
				}
			}
		}
		latest = f
		return stop(r.Context())
	})
	if err != nil || rows == 0 {
		fail(w, 404, "no readings with their full detail in that stretch")
		return
	}
	for k, n := range seen {
		if n != rows {
			changed[k] = true // present in some readings and not others
		}
	}
	var cols, constants []string
	for k := range first {
		if bookkeeping(k) {
			continue
		}
		if changed[k] {
			cols = append(cols, k)
		} else {
			constants = append(constants, k)
		}
	}
	for k := range changed { // a field that first appeared after the first reading
		if _, in := first[k]; !in && !bookkeeping(k) {
			cols = append(cols, k)
		}
	}
	sort.Strings(cols)
	sort.Strings(constants)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="cartrak-fields-%s.zip"`, time.Now().In(loc).Format("2006-01-02")))
	zw := zip.NewWriter(w)
	defer zw.Close()
	if f, err := zw.Create("README.txt"); err == nil {
		_, _ = f.Write([]byte(fieldsReadme))
	}
	write := func(name string, header []string, fill func(c *csv.Writer)) *csv.Writer {
		f, err := zw.Create(name)
		if err != nil {
			return nil
		}
		c := csv.NewWriter(f)
		_ = c.Write(header)
		fill(c)
		c.Flush()
		return c
	}
	write("latest-reading.csv", []string{"field", "value"}, func(c *csv.Writer) {
		keys := make([]string, 0, len(latest))
		for k := range latest {
			if keep(k) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			_ = c.Write([]string{k, fieldValue(latest[k])})
		}
	})
	write("unchanged.csv", []string{"field", "value"}, func(c *csv.Writer) {
		for _, k := range constants {
			_ = c.Write([]string{k, first[k]})
		}
	})

	// Second pass: a row each time anything in the changing columns differs from the row before.
	f, err := zw.Create("changes.csv")
	if err != nil {
		return
	}
	c := csv.NewWriter(f)
	_ = c.Write(append([]string{"time_utc", "time_local"}, cols...))
	var prev []string
	_ = s.S.EachExtra(from, to, func(ts int64, extra []byte) error {
		fl := map[string]any{}
		if json.Unmarshal(extra, &fl) != nil {
			return nil
		}
		row := make([]string, len(cols))
		for i, k := range cols {
			row[i] = fieldValue(fl[k])
		}
		if prev == nil || strings.Join(row, "\x00") != strings.Join(prev, "\x00") {
			t := time.Unix(ts, 0)
			_ = c.Write(append([]string{t.UTC().Format("2006-01-02T15:04:05Z"), t.In(loc).Format("2006-01-02 15:04:05")}, row...))
			prev = row
		}
		return stop(r.Context())
	})
	c.Flush()
}
