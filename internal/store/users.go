package store

import (
	"database/sql"
	"time"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Hash string `json:"-"`
	Role string `json:"role"` // admin | viewer
}

func (s *Store) CountUsers() int {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n
}

func (s *Store) CreateUser(name, hash, role string) error {
	_, err := s.DB.Exec(`INSERT INTO users(name,hash,role) VALUES(?,?,?)`, name, hash, role)
	return err
}

func (s *Store) UserByName(name string) (*User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id,name,hash,role FROM users WHERE name=?`, name).Scan(&u.ID, &u.Name, &u.Hash, &u.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(`SELECT id,name,role FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

func (s *Store) NewSession(token string, userID int64, ttl time.Duration) error {
	_, err := s.DB.Exec(`INSERT INTO sessions(token,user_id,expires) VALUES(?,?,?)`, token, userID, time.Now().Add(ttl).Unix())
	return err
}

func (s *Store) SessionUser(token string) (*User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT u.id,u.name,u.hash,u.role FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.token=? AND s.expires>?`, token, time.Now().Unix()).Scan(&u.ID, &u.Name, &u.Hash, &u.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (s *Store) DeleteSession(token string) {
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE token=?`, token)
}

// SetPassword changes an existing user's password hash and signs them out everywhere.
// It reports whether the user existed.
func (s *Store) SetPassword(name, hash string) (bool, error) {
	r, err := s.DB.Exec(`UPDATE users SET hash=? WHERE name=?`, hash, name)
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	if n > 0 {
		_, _ = s.DB.Exec(`DELETE FROM sessions WHERE user_id=(SELECT id FROM users WHERE name=?)`, name)
	}
	return n > 0, nil
}
