package store

import "time"

// Signin is one attempt to sign in, kept for the sign-in log.
type Signin struct {
	ID    int64  `json:"id"`
	TS    int64  `json:"ts"`
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	IP    string `json:"ip"`
	Agent string `json:"agent"`
}

// RecordSignin adds a line to the sign-in log and keeps the newest 500. The password is never stored.
func (s *Store) RecordSignin(name string, ok bool, ip, agent string) error {
	if _, err := s.DB.Exec(`INSERT INTO signins(ts,name,ok,ip,agent) VALUES(?,?,?,?,?)`, time.Now().Unix(), name, b2i(ok), ip, agent); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM signins WHERE id <= (SELECT MAX(id) FROM signins) - 500`)
	return err
}

func (s *Store) RecentSignins(limit int) ([]Signin, error) {
	rows, err := s.DB.Query(`SELECT id,ts,name,ok,ip,agent FROM signins ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Signin{}
	for rows.Next() {
		var x Signin
		var ok int
		if err := rows.Scan(&x.ID, &x.TS, &x.Name, &ok, &x.IP, &x.Agent); err != nil {
			return nil, err
		}
		x.OK = ok == 1
		out = append(out, x)
	}
	return out, rows.Err()
}

// SessionInfo is a signed-in browser or phone.
type SessionInfo struct {
	ID      int64  `json:"id"`
	User    string `json:"user"`
	Role    string `json:"role"`
	Created int64  `json:"created"`
	Expires int64  `json:"expires"`
	IP      string `json:"ip"`
	Agent   string `json:"agent"`
	Token   string `json:"-"`
}

// SetSessionInfo records where a session was started from.
func (s *Store) SetSessionInfo(token, ip, agent string) error {
	_, err := s.DB.Exec(`UPDATE sessions SET created=?,ip=?,agent=? WHERE token=?`, time.Now().Unix(), ip, agent, token)
	return err
}

// ListSessions returns the sessions that have not expired, newest first.
func (s *Store) ListSessions() ([]SessionInfo, error) {
	rows, err := s.DB.Query(`SELECT s.rowid,u.name,u.role,s.created,s.expires,s.ip,s.agent,s.token FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.expires>? ORDER BY s.created DESC, s.rowid DESC`, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SessionInfo{}
	for rows.Next() {
		var x SessionInfo
		if err := rows.Scan(&x.ID, &x.User, &x.Role, &x.Created, &x.Expires, &x.IP, &x.Agent, &x.Token); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSessionByID(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE rowid=?`, id)
	return err
}
