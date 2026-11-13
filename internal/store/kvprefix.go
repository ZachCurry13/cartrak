package store

import "strings"

// KVPair is one stored key and its value.
type KVPair struct{ K, V string }

// KVWithPrefix lists the keys that start with prefix (the remembered companies, "company:"), in key order.
func (s *Store) KVWithPrefix(prefix string) ([]KVPair, error) {
	esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(prefix)
	rows, err := s.DB.Query(`SELECT k, v FROM kv WHERE k LIKE ? ESCAPE '\' ORDER BY k`, esc+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KVPair
	for rows.Next() {
		var p KVPair
		if err := rows.Scan(&p.K, &p.V); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DeleteKV removes one key.
func (s *Store) DeleteKV(k string) error {
	_, err := s.DB.Exec(`DELETE FROM kv WHERE k=?`, k)
	return err
}
