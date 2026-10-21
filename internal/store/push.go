package store

import (
	"errors"
	"strings"
	"time"
)

// PushSub is one browser or phone that asked for notifications. The keys are what the push
// service needs to deliver an encrypted message to that device only.
type PushSub struct {
	ID       int64    `json:"id"`
	UserID   int64    `json:"userId"`
	Endpoint string   `json:"endpoint"`
	P256dh   string   `json:"-"`
	Auth     string   `json:"-"`
	Label    string   `json:"label"`
	Events   []string `json:"events"`
	Quiet    string   `json:"quiet"` // "22:00-07:00", or empty for no quiet hours
	Created  int64    `json:"created"`
	LastOK   int64    `json:"lastOk"`
	LastErr  string   `json:"lastErr"`
}

const pushCols = `id,user_id,endpoint,p256dh,auth,label,events,quiet,created,last_ok,last_err`

func scanSub(r interface{ Scan(...any) error }) (*PushSub, error) {
	var s PushSub
	var ev string
	if err := r.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.Label, &ev, &s.Quiet, &s.Created, &s.LastOK, &s.LastErr); err != nil {
		return nil, err
	}
	s.Events = []string{}
	for _, e := range strings.Split(ev, ",") {
		if e = strings.TrimSpace(e); e != "" {
			s.Events = append(s.Events, e)
		}
	}
	return &s, nil
}

// SavePushSub adds a device, or updates it if the same endpoint is already known (then it
// moves to the signed-in person and takes the new keys and choices).
func (s *Store) SavePushSub(p *PushSub) error {
	_, err := s.DB.Exec(`INSERT INTO push_subs(user_id,endpoint,p256dh,auth,label,events,quiet,created) VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(endpoint) DO UPDATE SET user_id=excluded.user_id,p256dh=excluded.p256dh,auth=excluded.auth,
		label=excluded.label,events=excluded.events,quiet=excluded.quiet,last_err=''`,
		p.UserID, p.Endpoint, p.P256dh, p.Auth, p.Label, strings.Join(p.Events, ","), p.Quiet, time.Now().Unix())
	return err
}

// ListPushSubs returns one person's devices, or everyone's when userID is 0.
func (s *Store) ListPushSubs(userID int64) ([]*PushSub, error) {
	q, a := `SELECT `+pushCols+` FROM push_subs`, []any{}
	if userID != 0 {
		q, a = q+` WHERE user_id=?`, append(a, userID)
	}
	rows, err := s.DB.Query(q+` ORDER BY id`, a...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*PushSub{}
	for rows.Next() {
		p, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPushSub(userID int64, endpoint string) (*PushSub, error) {
	return scanSub(s.DB.QueryRow(`SELECT `+pushCols+` FROM push_subs WHERE user_id=? AND endpoint=?`, userID, endpoint))
}

// SetPushPrefs changes which alerts a device gets and its quiet hours.
// ErrNoDevice is a device the server does not know (any more).
var ErrNoDevice = errors.New("no such device")

func (s *Store) SetPushPrefs(userID int64, endpoint string, events []string, quiet string) error {
	r, err := s.DB.Exec(`UPDATE push_subs SET events=?, quiet=? WHERE user_id=? AND endpoint=?`, strings.Join(events, ","), quiet, userID, endpoint)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return ErrNoDevice
	}
	return nil
}

func (s *Store) DeletePushSub(userID int64, endpoint string) error {
	_, err := s.DB.Exec(`DELETE FROM push_subs WHERE user_id=? AND endpoint=?`, userID, endpoint)
	return err
}

func (s *Store) DeletePushSubID(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM push_subs WHERE id=?`, id)
	return err
}

// MarkPush records how the last delivery went.
func (s *Store) MarkPush(id int64, errText string) error {
	if errText == "" {
		_, err := s.DB.Exec(`UPDATE push_subs SET last_ok=?, last_err='' WHERE id=?`, time.Now().Unix(), id)
		return err
	}
	_, err := s.DB.Exec(`UPDATE push_subs SET last_err=? WHERE id=?`, errText, id)
	return err
}
