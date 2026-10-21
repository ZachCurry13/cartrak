package store

import (
	"database/sql"
	"time"
)

type User struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Hash   string `json:"-"`
	Role   string `json:"role"`   // admin | viewer
	OwnCar bool   `json:"ownCar"` // may bring a car of their own
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
	var own int
	err := s.DB.QueryRow(`SELECT id,name,hash,role,own_car FROM users WHERE name=?`, name).Scan(&u.ID, &u.Name, &u.Hash, &u.Role, &own)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.OwnCar = own != 0
	return &u, err
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(`SELECT id,name,role,own_car FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		var own int
		if err := rows.Scan(&u.ID, &u.Name, &u.Role, &own); err != nil {
			return nil, err
		}
		u.OwnCar = own != 0
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id=?`, id)
	_, _ = s.DB.Exec(`DELETE FROM car_access WHERE user_id=?`, id)
	s.forgetPrefs(id)
	return err
}

func (s *Store) NewSession(token string, userID int64, ttl time.Duration) error {
	_, err := s.DB.Exec(`INSERT INTO sessions(token,user_id,expires) VALUES(?,?,?)`, token, userID, time.Now().Add(ttl).Unix())
	return err
}

func (s *Store) SessionUser(token string) (*User, error) {
	var u User
	var own int
	err := s.DB.QueryRow(`SELECT u.id,u.name,u.hash,u.role,u.own_car FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.token=? AND s.expires>?`, token, time.Now().Unix()).Scan(&u.ID, &u.Name, &u.Hash, &u.Role, &own)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	u.OwnCar = own != 0
	if err == nil { // when it was last used, for the sign-ins page: written at most every five minutes
		now := time.Now().Unix()
		_, _ = s.DB.Exec(`UPDATE sessions SET last_active=? WHERE token=? AND last_active<?`, now, token, now-300)
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
