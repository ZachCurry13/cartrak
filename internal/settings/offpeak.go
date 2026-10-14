package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// Time-of-use pricing at home: a cheaper price for a window of the day, for example 22:00-06:00.
const (
	OffPeakPriceKey  = "CARTRAK_PRICE_OFFPEAK"
	OffPeakWindowKey = "CARTRAK_OFFPEAK_WINDOW"
)

func init() {
	Defs = append(Defs,
		Def{Key: OffPeakPriceKey, Label: "Cheaper home price per kWh (optional)", Placeholder: "0.08",
			Help: "If your electricity plan has cheaper hours, the price in those hours. Home charging in that window is priced with it, and the charging forecast shows what waiting for it would save. Leave empty if you have one price all day."},
		Def{Key: OffPeakWindowKey, Label: "Cheaper hours", Placeholder: "22:00-06:00",
			Help: "When the cheaper price applies, in your time zone, as start-end in 24-hour time. It may cross midnight, as in 22:00-06:00."})
	validators[OffPeakPriceKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err != nil || n < 0 || n > 20 {
			return fmt.Errorf("enter a price per kWh such as 0.08")
		}
		return nil
	}
	validators[OffPeakWindowKey] = func(v string) error {
		if v == "" {
			return nil
		}
		if _, _, ok := ParseWindow(v); !ok {
			return fmt.Errorf("use the form 22:00-06:00")
		}
		return nil
	}
}

// ParseWindow reads "22:00-06:00" as minutes after midnight. The two times must differ.
func ParseWindow(v string) (from, to int, ok bool) {
	a, b, found := strings.Cut(strings.TrimSpace(v), "-")
	if !found {
		return 0, 0, false
	}
	hm := func(s string) (int, bool) {
		h, m, f := strings.Cut(strings.TrimSpace(s), ":")
		if !f || len(m) != 2 {
			return 0, false
		}
		hh, e1 := strconv.Atoi(h)
		mm, e2 := strconv.Atoi(m)
		if e1 != nil || e2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
			return 0, false
		}
		return hh*60 + mm, true
	}
	f, ok1 := hm(a)
	t, ok2 := hm(b)
	if !ok1 || !ok2 || f == t {
		return 0, 0, false
	}
	return f, t, true
}

// OffPeak is the cheaper home price and the window it applies in, or nils when not set up.
func (s *Settings) OffPeak() (price *float64, from, to int) {
	n, err := strconv.ParseFloat(strings.TrimSpace(s.Value(OffPeakPriceKey)), 64)
	if err != nil || n < 0 {
		return nil, 0, 0
	}
	f, t, ok := ParseWindow(s.Value(OffPeakWindowKey))
	if !ok {
		return nil, 0, 0
	}
	return &n, f, t
}
