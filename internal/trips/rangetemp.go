package trips

import "github.com/ZachCurry13/cartrak/internal/store"

// TempBand is a stretch of outside temperatures in degrees Celsius: Lo <= temperature < Hi.
type TempBand struct {
	Name   string  `json:"name"`
	Lo, Hi float64 `json:"-"`
}

// TempBands are the weather groups the range is worked out for.
var TempBands = []TempBand{{"Below freezing", -100, 0}, {"0 to 10 °C", 0, 10}, {"10 to 20 °C", 10, 20}, {"20 °C and above", 20, 100}}

// BandFor is the group a temperature falls in.
func BandFor(tempC float64) *TempBand {
	for i := range TempBands {
		if tempC >= TempBands[i].Lo && tempC < TempBands[i].Hi {
			return &TempBands[i]
		}
	}
	return nil
}

// BandWindow is the driving in one weather group.
type BandWindow struct {
	Band TempBand
	Window
}

// MeasureByBand groups trips by the average outside temperature during each (temps maps a trip id to it) and
// measures each group. Trips with no temperature are left out.
func MeasureByBand(ts []*store.Trip, temps map[int64]float64) []BandWindow {
	groups := make([][]*store.Trip, len(TempBands))
	for _, t := range ts {
		if temp, ok := temps[t.ID]; ok {
			if b := BandFor(temp); b != nil {
				for i := range TempBands {
					if TempBands[i].Name == b.Name {
						groups[i] = append(groups[i], t)
					}
				}
			}
		}
	}
	out := make([]BandWindow, 0, len(TempBands))
	for i, g := range groups {
		out = append(out, BandWindow{Band: TempBands[i], Window: Measure(g)})
	}
	return out
}
