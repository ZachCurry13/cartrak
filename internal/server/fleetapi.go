package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// What the fleet needs to know to send a request to the right car.

// CookieName is the sign-in cookie.
const CookieName = cookieName

// CarCookie remembers which car a browser last looked at.
const CarCookie = "cartrak_car"

// DeviceToken is the phone token a request carries, if any.
func DeviceToken(r *http.Request) string { return deviceToken(r) }

// HasDevice says whether a phone token belongs to a device of this server's car.
func (s *Server) HasDevice(token string) bool {
	if token == "" {
		return false
	}
	d, _ := s.S.DeviceByHash(store.TokenHash(token))
	return d != nil
}

// HasShare says whether a public share link belongs to this server's car.
func (s *Server) HasShare(token string) bool {
	sh, _ := s.S.ShareByHash(store.TokenHash(token))
	return sh != nil
}

// fordState is the value sent to Ford when linking: it starts with the car's number, so the page Ford sends the browser
// back to can be answered by that car. Ford allows at most 16 characters.
func (s *Server) fordState() string {
	if s.CarID == 0 {
		return randHex(8)
	}
	return fmt.Sprintf("%dx%s", s.CarID, randHex(6))
}

// CarFromState reads the car number out of a state made by fordState; 0 when there is none.
func CarFromState(state string) int64 {
	n, _, ok := strings.Cut(state, "x")
	if !ok {
		return 0
	}
	id, err := strconv.ParseInt(n, 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}
