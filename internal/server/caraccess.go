package server

import (
	"net/http"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/store"
)

type accessRow struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Level  string `json:"level"`
}

// carAccess: GET /api/cars/{id}/access. Who can open the car.
func (s *Server) carAccess(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if !s.manage(r, id) {
		fail(w, 403, "only the car's owner or an administrator can see who has access")
		return
	}
	m, _ := s.core().CarAccess(id)
	users, _ := s.core().ListUsers()
	out := []accessRow{}
	for _, u := range users {
		out = append(out, accessRow{UserID: u.ID, Name: u.Name, Level: m[u.ID]})
	}
	ok(w, out)
}

// setCarAccess: PUT /api/cars/{id}/access {userId, level} with level owner, viewer or "" (none). An owner can share the
// car with others as viewers; making someone an owner, or taking an owner's access away, is for site administrators.
func (s *Server) setCarAccess(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if !s.manage(r, id) {
		fail(w, 403, "only the car's owner or an administrator can change who has access")
		return
	}
	var b struct {
		UserID int64  `json:"userId"`
		Level  string `json:"level"`
	}
	if decode(r, &b) != nil || (b.Level != "" && b.Level != store.AccessOwner && b.Level != store.AccessViewer) {
		fail(w, 400, "level must be owner, viewer or empty")
		return
	}
	u := userFrom(r)
	if u.Role != "admin" && (b.Level == store.AccessOwner || s.core().Access(b.UserID, id) == store.AccessOwner) {
		fail(w, 403, "only an administrator can make or remove an owner")
		return
	}
	if car, _ := s.core().GetCar(id); car == nil {
		fail(w, 404, "no such car")
		return
	}
	if err := s.core().SetAccess(b.UserID, id, b.Level); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "saved")
}

// patchUser: PATCH /api/users/{id} {ownCar, seeMain}. A site administrator sets what a person may do with cars.
// seeMain gives or takes away read access to the first car.
func (s *Server) patchUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var b struct {
		OwnCar  *bool `json:"ownCar"`
		SeeMain *bool `json:"seeMain"`
	}
	if decode(r, &b) != nil {
		fail(w, 400, "bad request")
		return
	}
	if b.OwnCar != nil {
		if err := s.core().SetOwnCar(id, *b.OwnCar); err != nil {
			fail(w, 500, "db error")
			return
		}
	}
	if b.SeeMain != nil {
		cars, _ := s.core().ListCars()
		if len(cars) > 0 && s.core().Access(id, cars[0].ID) != store.AccessOwner {
			level := ""
			if *b.SeeMain {
				level = store.AccessViewer
			}
			_ = s.core().SetAccess(id, cars[0].ID, level)
		}
	}
	ok(w, "saved")
}

// grantNewUser gives a new person the starting access: read access to the first car (administrators own it).
func (s *Server) grantNewUser(name, role string, seeMain bool) {
	u, _ := s.core().UserByName(name)
	cars, _ := s.core().ListCars()
	if u == nil || len(cars) == 0 || s.CarID == 0 {
		return
	}
	switch {
	case role == "admin":
		_ = s.core().SetAccess(u.ID, cars[0].ID, store.AccessOwner)
	case seeMain:
		_ = s.core().SetAccess(u.ID, cars[0].ID, store.AccessViewer)
	}
}
