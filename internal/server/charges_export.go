package server

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ZachCurry13/cartrak/internal/charging"
)

// exportChargesCSV is every charging session as a spreadsheet: energy added, what the charger reported, the
// efficiency where known, where it was, and the cost.
func (s *Server) exportChargesCSV(w http.ResponseWriter, r *http.Request) {
	ss, err := s.chargeSessions(1000)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="cartrak-charging.csv"`)
	s.writeChargesCSV(w, ss)
}

func (s *Server) writeChargesCSV(w io.Writer, ss []charging.Session) {
	c := csv.NewWriter(w)
	_ = c.Write([]string{"id", "start", "end", "start_local", "start_percent", "end_percent", "added_kwh", "added_estimated", "charger_kwh", "efficiency_percent", "where", "peak_kw", "cost", "cost_from_receipt", "charger_type"})
	loc := s.defaultLoc()
	for _, x := range ss {
		end := ""
		if x.EndTS != nil {
			end = time.Unix(*x.EndTS, 0).UTC().Format(time.RFC3339)
		}
		eff := ""
		if x.Efficiency != nil {
			eff = fmt.Sprintf("%.1f", *x.Efficiency*100)
		}
		_ = c.Write([]string{strconv.FormatInt(x.ID, 10), time.Unix(x.StartTS, 0).UTC().Format(time.RFC3339), end,
			time.Unix(x.StartTS, 0).In(loc).Format("2006-01-02 15:04"), fl(x.StartSoC), fl(x.EndSoC), fl2(x.AddedKwh), strconv.FormatBool(x.Estimated),
			fl2(x.ChargerKwh), eff, x.Where, fmt.Sprintf("%.1f", x.PeakKw), fl2(x.Cost), strconv.FormatBool(x.CostActual), x.Station})
	}
	c.Flush()
}
