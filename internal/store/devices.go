package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strconv"
	"time"
)

// Device is a phone (or other gadget) allowed to send tracking data with a token.
// Only a hash of the token is stored, so it cannot be read back.
type Device struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Created   int64  `json:"created"`
	LastSeen  int64  `json:"lastSeen"`
	LastEvent string `json:"lastEvent"`
	Points    int    `json:"points"`
}

// TokenHash is what is kept in the database for a device token.
func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *Store) CreateDevice(name, tokenHash string) (int64, error) {
	r, err := s.DB.Exec(`INSERT INTO devices(name,token_hash,created) VALUES(?,?,?)`, name, tokenHash, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// DeviceByHash finds the device that owns a token hash, or nil.
func (s *Store) DeviceByHash(hash string) (*Device, error) {
	var d Device
	err := s.DB.QueryRow(`SELECT id,name,created,last_seen,last_event FROM devices WHERE token_hash=?`, hash).
		Scan(&d.ID, &d.Name, &d.Created, &d.LastSeen, &d.LastEvent)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

func (s *Store) ListDevices() ([]Device, error) {
	rows, err := s.DB.Query(`SELECT d.id,d.name,d.created,d.last_seen,d.last_event,
	  (SELECT COUNT(*) FROM track_points p WHERE p.device_id=d.id) FROM devices d ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Device{}
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Created, &d.LastSeen, &d.LastEvent, &d.Points); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteDevice removes the device and so its token. Points it already sent are kept.
func (s *Store) DeleteDevice(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM devices WHERE id=?`, id)
	return err
}

// TouchDevice records that the device was heard from, and what it last said.
func (s *Store) TouchDevice(id int64, event string) error {
	if event == "" {
		_, err := s.DB.Exec(`UPDATE devices SET last_seen=? WHERE id=?`, time.Now().Unix(), id)
		return err
	}
	_, err := s.DB.Exec(`UPDATE devices SET last_seen=?, last_event=? WHERE id=?`, time.Now().Unix(), event, id)
	return err
}

// BrowserDevice is the stand-in "device" for a signed-in browser that records GPS from the
// Live page. Its token hash is not valid hex, so no bearer token can ever match it.
func (s *Store) BrowserDevice(userID int64, who string) (*Device, error) {
	key := "browser:" + strconv.FormatInt(userID, 10)
	if d, err := s.DeviceByHash(key); d != nil || err != nil {
		return d, err
	}
	if _, err := s.CreateDevice("Browser ("+who+")", key); err != nil {
		return nil, err
	}
	return s.DeviceByHash(key)
}
