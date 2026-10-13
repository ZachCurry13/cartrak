package store

import (
	"database/sql"
	"encoding/json"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

// SaveState stores the raw response and the parsed sample; returns the sample id.
func (s *Store) SaveState(provider string, st *ford.State) (int64, error) {
	r, err := s.DB.Exec(`INSERT INTO snapshots(ts,provider,raw) VALUES(?,?,?)`,
		st.TS.Unix(), provider, string(st.Raw))
	if err != nil {
		return 0, err
	}
	snap, _ := r.LastInsertId()
	extra, _ := json.Marshal(st.Extra)
	var locked any
	if st.Locked != nil {
		locked = *st.Locked
	}
	r, err = s.DB.Exec(`INSERT INTO samples(ts,snapshot_id,odometer_km,soc_pct,range_km,lat,lon,
		speed_kph,ignition,charge_state,plug,charge_kw,outside_c,battery12v_v,
		tire_fl,tire_fr,tire_rl,tire_rr,locked,extra,fuel_pct,oil_pct,hv_volts,capacity_kwh) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		st.TS.Unix(), snap, st.OdometerKm, st.SoCPct, st.RangeKm, st.Lat, st.Lon,
		st.SpeedKph, st.Ignition, st.ChargeState, st.Plug, st.ChargeKw, st.OutsideC, st.Battery12V,
		st.Tires[0], st.Tires[1], st.Tires[2], st.Tires[3], locked, string(extra),
		st.FuelPct, st.OilLifePct, st.HVVolts, st.CapacityKwh)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	_ = s.saveTrend(id, st) // the charted numbers, kept apart from the big row
	return id, nil
}

// Sample is one stored reading as served to the UI.
type Sample struct {
	TS          int64           `json:"ts"`
	OdometerKm  *float64        `json:"odometerKm"`
	SoCPct      *float64        `json:"socPct"`
	RangeKm     *float64        `json:"rangeKm"`
	Lat         *float64        `json:"lat"`
	Lon         *float64        `json:"lon"`
	SpeedKph    *float64        `json:"speedKph"`
	Ignition    string          `json:"ignition"`
	ChargeState string          `json:"chargeState"`
	Plug        string          `json:"plug"`
	ChargeKw    *float64        `json:"chargeKw"`
	OutsideC    *float64        `json:"outsideC"`
	Battery12V  *float64        `json:"battery12v"`
	Tires       [4]*float64     `json:"tires"`
	Locked      *bool           `json:"locked"`
	FuelPct     *float64        `json:"fuelPct"`
	OilLifePct  *float64        `json:"oilLifePct"`
	HVVolts     *float64        `json:"hvVolts"`
	CapacityKwh *float64        `json:"capacityKwh"`
	Extra       json.RawMessage `json:"extra"`
}

const sampleCols = `ts,odometer_km,soc_pct,range_km,lat,lon,speed_kph,ignition,charge_state,plug,
 charge_kw,outside_c,battery12v_v,tire_fl,tire_fr,tire_rl,tire_rr,locked,extra,fuel_pct,oil_pct,hv_volts,capacity_kwh`

func scanSample(r interface{ Scan(...any) error }) (*Sample, error) {
	var x Sample
	var ign, cs, plug, extra sql.NullString
	var locked sql.NullBool
	err := r.Scan(&x.TS, &x.OdometerKm, &x.SoCPct, &x.RangeKm, &x.Lat, &x.Lon, &x.SpeedKph,
		&ign, &cs, &plug, &x.ChargeKw, &x.OutsideC, &x.Battery12V,
		&x.Tires[0], &x.Tires[1], &x.Tires[2], &x.Tires[3], &locked, &extra,
		&x.FuelPct, &x.OilLifePct, &x.HVVolts, &x.CapacityKwh)
	if err != nil {
		return nil, err
	}
	x.Ignition, x.ChargeState, x.Plug = ign.String, cs.String, plug.String
	if locked.Valid {
		x.Locked = &locked.Bool
	}
	x.Extra = json.RawMessage(extra.String)
	if len(x.Extra) == 0 {
		x.Extra = json.RawMessage("{}")
	}
	return &x, nil
}

// LatestSample returns the newest reading, or nil if there is none yet.
func (s *Store) LatestSample() (*Sample, error) {
	x, err := scanSample(s.DB.QueryRow(`SELECT ` + sampleCols + ` FROM samples ORDER BY ts DESC LIMIT 1`))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return x, err
}

// SamplesBetween returns readings in [from,to] oldest first (for trip maps/charts).
func (s *Store) SamplesBetween(from, to int64) ([]*Sample, error) {
	rows, err := s.DB.Query(`SELECT `+sampleCols+` FROM samples WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Sample
	for rows.Next() {
		x, err := scanSample(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// TrackBetween is SamplesBetween without the large raw fields: just what a route needs (time,
// position, speed). Use it wherever the other values are not shown; the full rows are slow to
// read in bulk.
func (s *Store) TrackBetween(from, to int64) ([]*Sample, error) {
	rows, err := s.DB.Query(`SELECT ts,lat,lon,speed_kph FROM samples WHERE ts BETWEEN ? AND ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Sample
	for rows.Next() {
		x := &Sample{Extra: json.RawMessage("{}")}
		if err := rows.Scan(&x.TS, &x.Lat, &x.Lon, &x.SpeedKph); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
