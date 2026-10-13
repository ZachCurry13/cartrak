package server

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// deviceToken reads the device token from either "Authorization: Bearer <token>" or HTTP Basic
// (the token is the password and the user name is ignored), which is what apps such as OwnTracks send.
func deviceToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	if b, ok := strings.CutPrefix(h, "Basic "); ok {
		if raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b)); err == nil {
			if _, pw, ok := strings.Cut(string(raw), ":"); ok {
				return strings.TrimSpace(pw)
			}
		}
	}
	return ""
}

// ingestOwnTracks takes the OwnTracks app's HTTP posts. Only {"_type":"location"} messages are
// used (OwnTracks also sends other kinds, which are acknowledged and ignored). The app expects a
// JSON array back; an empty one means "nothing for you". Speed (vel) arrives in km/h.
func (s *Server) ingestOwnTracks(w http.ResponseWriter, r *http.Request, d *store.Device) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		fail(w, 400, "could not read the request")
		return
	}
	nothing := func() { ok(w, []any{}) }
	if strings.TrimSpace(string(body)) == "" { // the app sometimes sends an empty message
		nothing()
		return
	}
	var in struct {
		Type string `json:"_type"`
		Lat  *num   `json:"lat"`
		Lon  *num   `json:"lon"`
		Tst  int64  `json:"tst"`
		Vel  *num   `json:"vel"`
		Acc  *num   `json:"acc"`
	}
	if jsonDecode(body, &in) != nil || in.Type != "location" {
		nothing()
		return
	}
	// Reuse the same checks as every other source: sensible position and time.
	f := fixReq{TS: in.Tst, Lat: in.Lat, Lon: in.Lon, Acc: in.Acc}
	p, good := f.point()
	if !good {
		nothing()
		return
	}
	if in.Vel != nil && *in.Vel >= 0 {
		kph := float64(*in.Vel)
		p.SpeedKph = &kph
	}
	if err := s.S.SavePoints(d.ID, []store.Point{p}); err != nil {
		fail(w, 500, "could not save the location")
		return
	}
	_ = s.S.TouchDevice(d.ID, "")
	if s.P != nil && movingNow([]store.Point{p}) {
		s.P.PhoneDriving(true)
	}
	nothing()
}
