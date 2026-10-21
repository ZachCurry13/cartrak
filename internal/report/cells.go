package report

import (
	"fmt"

	"github.com/ZachCurry13/cartrak/internal/xlsx"
)

// cell is what one column shows for one trip.
func (o Options) cell(key string, t Trip) xlsx.Cell {
	n := func(v float64, d int) xlsx.Cell { return xlsx.N(round(v, d)) }
	np := func(p *float64, d int) xlsx.Cell {
		if p == nil {
			return xlsx.S("")
		}
		return n(*p, d)
	}
	switch key {
	case "value":
		return n(o.value(t), 2)
	case "distance":
		return n(o.dist(t.DistanceKm), 1)
	case "rate":
		return n(o.rate(t.Category), 3)
	case "from":
		return xlsx.S(t.From)
	case "to":
		return xlsx.S(t.To)
	case "date":
		return xlsx.S(t.Start.Format("2006-01-02"))
	case "purpose":
		if l := o.Labels[t.Category]; l != "" {
			return xlsx.S(l)
		}
		return xlsx.S(t.Category)
	case "track":
		return xlsx.S(t.Method)
	case "notes":
		return xlsx.S(t.Notes)
	case "vehicle":
		return xlsx.S(o.Vehicle)
	case "started":
		return xlsx.S(t.Start.Format("03:04 PM"))
	case "ended":
		return xlsx.S(t.End.Format("03:04 PM"))
	case "hours":
		return n(t.End.Sub(t.Start).Hours(), 1)
	case "zone":
		return xlsx.S(t.Zone)
	case "weekday":
		return xlsx.S(t.Start.Weekday().String())
	case "energy":
		return np(t.EnergyKwh, 2)
	case "efficiency":
		if t.EnergyKwh == nil || *t.EnergyKwh <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return xlsx.S(o.effText(t.DistanceKm, *t.EnergyKwh))
	case "energycost":
		if t.EnergyKwh == nil || o.KwhPrice <= 0 {
			return xlsx.S("")
		}
		return n(*t.EnergyKwh*o.KwhPrice, 2)
	case "costper":
		if t.EnergyKwh == nil || o.KwhPrice <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return n(*t.EnergyKwh*o.KwhPrice/o.dist(t.DistanceKm), 3)
	case "mpge":
		if t.EnergyKwh == nil || *t.EnergyKwh <= 0 || t.DistanceKm < 0.5 {
			return xlsx.S("")
		}
		return n(t.DistanceKm/kmPerMile/(*t.EnergyKwh/33.705), 0) // 33.705 kWh is the energy of a gallon of gasoline
	case "avgspeed":
		if h := t.End.Sub(t.Start).Hours(); h > 0 {
			return n(o.dist(t.DistanceKm)/h, 0)
		}
		return xlsx.S("")
	case "socstart":
		return np(t.SoCStart, 0)
	case "socend":
		return np(t.SoCEnd, 0)
	case "odostart":
		if t.OdoStartKm == nil {
			return xlsx.S("")
		}
		return n(o.dist(*t.OdoStartKm), 1)
	case "odoend":
		if t.OdoEndKm == nil {
			return xlsx.S("")
		}
		return n(o.dist(*t.OdoEndKm), 1)
	case "topspeed":
		return n(o.dist(t.TopSpeedKph), 0)
	case "id":
		return xlsx.N(float64(t.ID))
	}
	return xlsx.S("")
}

// effText writes how far a trip went on its energy, in the unit chosen: miles per kWh, watt-hours per mile, or miles per percent of
// the battery (kilometers when metric; kWh per 100 km stands in for the first, as the app does).
func (o Options) effText(km, kwh float64) string {
	switch {
	case o.EffUnit == "pct" && o.Capacity > 0:
		used := kwh / (o.Capacity / 100)
		if o.Metric {
			return fmt.Sprintf("%.2f km/%%", km/used)
		}
		return fmt.Sprintf("%.2f mi/%%", km/kmPerMile/used)
	case o.EffUnit == "wh":
		if o.Metric {
			return fmt.Sprintf("%d Wh/km", int(kwh*1000/km+0.5))
		}
		return fmt.Sprintf("%d Wh/mi", int(kwh*1000/(km/kmPerMile)+0.5))
	case o.Metric:
		return fmt.Sprintf("%.1f kWh/100 km", kwh/km*100)
	}
	return fmt.Sprintf("%.2f mi/kWh", km/kmPerMile/kwh)
}
