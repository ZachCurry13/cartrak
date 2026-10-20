package store

import "database/sql"

// migrations add columns to tables created by an older version. Each runs only
// if the column is missing, so it is safe on new and old databases alike.
var migrations = []struct{ table, column, ddl string }{
	{"trips", "tz", `ALTER TABLE trips ADD COLUMN tz TEXT NOT NULL DEFAULT ''`},
	{"samples", "fuel_pct", `ALTER TABLE samples ADD COLUMN fuel_pct REAL`},
	{"samples", "oil_pct", `ALTER TABLE samples ADD COLUMN oil_pct REAL`},
	{"samples", "hv_volts", `ALTER TABLE samples ADD COLUMN hv_volts REAL`},
	{"samples", "capacity_kwh", `ALTER TABLE samples ADD COLUMN capacity_kwh REAL`},
	{"samples", "energy_kwh", `ALTER TABLE samples ADD COLUMN energy_kwh REAL`},
	{"trends", "range_km", `ALTER TABLE trends ADD COLUMN range_km REAL`},
	{"trends", "odometer_km", `ALTER TABLE trends ADD COLUMN odometer_km REAL`},
	{"trends", "speed_kph", `ALTER TABLE trends ADD COLUMN speed_kph REAL`},
	{"trends", "outside_c", `ALTER TABLE trends ADD COLUMN outside_c REAL`},
	{"trends", "driving", `ALTER TABLE trends ADD COLUMN driving INTEGER NOT NULL DEFAULT 0`},
	{"trends", "charging", `ALTER TABLE trends ADD COLUMN charging INTEGER NOT NULL DEFAULT 0`},
	{"trips", "start_kwh", `ALTER TABLE trips ADD COLUMN start_kwh REAL`},
	{"trips", "end_kwh", `ALTER TABLE trips ADD COLUMN end_kwh REAL`},
	{"sessions", "created", `ALTER TABLE sessions ADD COLUMN created INTEGER NOT NULL DEFAULT 0`},
	{"sessions", "ip", `ALTER TABLE sessions ADD COLUMN ip TEXT NOT NULL DEFAULT ''`},
	{"sessions", "agent", `ALTER TABLE sessions ADD COLUMN agent TEXT NOT NULL DEFAULT ''`},
	{"trips", "journey_id", `ALTER TABLE trips ADD COLUMN journey_id INTEGER`},
	{"charges", "start_kwh", `ALTER TABLE charges ADD COLUMN start_kwh REAL`},
	{"charges", "end_kwh", `ALTER TABLE charges ADD COLUMN end_kwh REAL`},
	{"charges", "charger_kwh", `ALTER TABLE charges ADD COLUMN charger_kwh REAL`},
	{"charges", "kind", `ALTER TABLE charges ADD COLUMN kind TEXT NOT NULL DEFAULT ''`},
	{"charges", "actual_cost", `ALTER TABLE charges ADD COLUMN actual_cost REAL`},
	{"charges", "power_kwh", `ALTER TABLE charges ADD COLUMN power_kwh REAL`},
	{"users", "own_car", `ALTER TABLE users ADD COLUMN own_car INTEGER NOT NULL DEFAULT 0`},
	{"charges", "station", `ALTER TABLE charges ADD COLUMN station TEXT NOT NULL DEFAULT ''`},
}

func migrate(db *sql.DB) error {
	for _, m := range migrations {
		has, err := hasColumn(db, m.table, m.column)
		if err != nil {
			return err
		}
		if !has {
			if _, err := db.Exec(m.ddl); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return false, err
		}
		if n == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
