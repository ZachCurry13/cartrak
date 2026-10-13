package store

import "database/sql"

// migrations add columns to tables created by an older version. Each runs only
// if the column is missing, so it is safe on new and old databases alike.
var migrations = []struct{ table, column, ddl string }{
	{"trips", "tz", `ALTER TABLE trips ADD COLUMN tz TEXT NOT NULL DEFAULT ''`},
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
