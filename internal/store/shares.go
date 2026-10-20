package store

import (
	"database/sql"
	"time"
)

// Share is a public link to one trip or journey. Only a hash of the link's secret is kept, so the link can
// be shown once and never read back.
type Share struct {
	ID         int64  `json:"id"`
	Kind       string `json:"kind"` // trip or journey
	TargetID   int64  `json:"targetId"`
	Title      string `json:"title"`
	Mode       string `json:"mode"` // stats, trimmed or full
	ShowTime   bool   `json:"showTime"`
	Units      string `json:"units"` // mi or km
	Origin     string `json:"origin"`
	Dest       string `json:"dest"`
	BBox       string `json:"-"` // minLat,minLon,maxLat,maxLon of the shared route, for the map tiles
	Created    int64  `json:"created"`
	Expires    int64  `json:"expires"` // 0 never
	Views      int    `json:"views"`
	LastView   int64  `json:"lastView"`
	Revoked    bool   `json:"revoked"`
	TokenHash  string `json:"-"`
	ShowCities bool   `json:"showCities"`
	// ShowInsights adds the efficiency details to the public page: average and top speed, the temperature and what made the trip use more or
	// less energy.
	ShowInsights bool `json:"showInsights"`
}

const shareCols = `id,kind,target_id,title,mode,show_time,units,origin,dest,bbox,created,expires,views,last_view,revoked,token_hash,show_cities,show_insights`

func scanShare(r interface{ Scan(...any) error }) (*Share, error) {
	var s Share
	var st, rv, sc, si int
	err := r.Scan(&s.ID, &s.Kind, &s.TargetID, &s.Title, &s.Mode, &st, &s.Units, &s.Origin, &s.Dest, &s.BBox, &s.Created, &s.Expires, &s.Views, &s.LastView, &rv, &s.TokenHash, &sc, &si)
	s.ShowTime, s.Revoked, s.ShowCities, s.ShowInsights = st == 1, rv == 1, sc == 1, si == 1
	return &s, err
}

func (s *Store) CreateShare(x *Share) error {
	r, err := s.DB.Exec(`INSERT INTO shares(`+shareCols[len("id,"):]+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0,?,?,?)`,
		x.Kind, x.TargetID, x.Title, x.Mode, b2i(x.ShowTime), x.Units, x.Origin, x.Dest, x.BBox, time.Now().Unix(), x.Expires, 0, 0, x.TokenHash, b2i(x.ShowCities), b2i(x.ShowInsights))
	if err != nil {
		return err
	}
	x.ID, _ = r.LastInsertId()
	return nil
}

// ShareByHash finds a share that is still valid: not revoked and not expired.
func (s *Store) ShareByHash(hash string) (*Share, error) {
	x, err := scanShare(s.DB.QueryRow(`SELECT `+shareCols+` FROM shares WHERE token_hash=? AND revoked=0 AND (expires=0 OR expires>?)`, hash, time.Now().Unix()))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return x, err
}

func (s *Store) ListShares() ([]*Share, error) {
	rows, err := s.DB.Query(`SELECT ` + shareCols + ` FROM shares ORDER BY id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Share{}
	for rows.Next() {
		x, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) RevokeShare(id int64) error {
	_, err := s.DB.Exec(`UPDATE shares SET revoked=1 WHERE id=?`, id)
	return err
}

func (s *Store) CountShareView(id int64) {
	_, _ = s.DB.Exec(`UPDATE shares SET views=views+1,last_view=? WHERE id=?`, time.Now().Unix(), id)
}
