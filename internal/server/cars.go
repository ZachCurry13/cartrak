package server

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// CarManager is what the server asks of the fleet to add, rename and remove cars. It is nil when only one car exists
// (tests), and then the cars API says it is not available.
type CarManager interface {
	AddCar(name string, ownerID int64) (int64, error)
	Rename(id int64, name string) error
	Remove(id int64) error
	Status(id int64) (linked bool, lastOK int64, found bool)
	BackupAll(tmpDir string, w io.Writer) error           // a zip of the core database and every car's database
	Inbox(userID int64, limit int) []InboxItem            // the latest alerts about the cars a person can open
	AllDebugReports(hours int, metric bool) (string, int) // every car's debug report in one text, and the readings they cover
	AdminViews(id int64) map[string]int64                 // administrators who opened the car without having been given access, by name, when last
}

// carView is one car as a person sees it in the car switcher.
type carView struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Level   string `json:"level"`   // owner, viewer, or admin (a site administrator looking at someone else's car)
	Theirs  bool   `json:"theirs"`  // true when the person has no access of their own and sees it as an administrator
	OwnerID int64  `json:"ownerId"` // who brought the car; 0 for the first
	Linked  bool   `json:"linked"`
	LastOK  int64  `json:"lastOk"`
	// AdminViews is shown to the car's owner: which administrators opened the car without having been given access.
	AdminViews map[string]int64 `json:"adminViews,omitempty"`
}

// listCars: GET /api/cars. The cars the signed-in person can open, with what they may do, and whether they may add one.
func (s *Server) listCars(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	cars, err := s.core().ListCars()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	access, _ := s.core().UserAccess(u.ID)
	out := []carView{}
	for _, c := range cars {
		lvl, mine := access[c.ID]
		if !mine && u.Role != "admin" {
			continue
		}
		v := carView{ID: c.ID, Name: c.Name, Level: lvl, Theirs: !mine, OwnerID: c.OwnerID}
		if !mine {
			v.Level = "admin"
		}
		if s.Cars != nil {
			v.Linked, v.LastOK, _ = s.Cars.Status(c.ID)
			if mine && lvl == store.AccessOwner {
				v.AdminViews = s.Cars.AdminViews(c.ID)
			}
		}
		out = append(out, v)
	}
	ok(w, map[string]any{"cars": out, "canAdd": s.Cars != nil && (u.Role == "admin" || u.OwnCar) && len(cars) < maxCars, "max": maxCars})
}

// maxCars matches fleet.MaxCars (the server cannot import the fleet).
const maxCars = 6

func cleanCarName(n string) string {
	n = strings.Join(strings.Fields(n), " ")
	if len([]rune(n)) > 40 {
		n = string([]rune(n)[:40])
	}
	return n
}

// addCar: POST /api/cars {name}. A site administrator, or a person allowed to bring a car, adds a car that they own.
func (s *Server) addCar(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if s.Cars == nil {
		fail(w, 501, "several cars are not available here")
		return
	}
	if u.Role != "admin" && !u.OwnCar {
		fail(w, 403, "you were not given the right to add a car; ask an administrator")
		return
	}
	var b struct{ Name string }
	if decode(r, &b) != nil || cleanCarName(b.Name) == "" {
		fail(w, 400, "give the car a name")
		return
	}
	if cars, _ := s.core().ListCars(); len(cars) >= maxCars {
		fail(w, 409, "that is the most cars CarTrak follows ("+strconv.Itoa(maxCars)+")")
		return
	}
	id, err := s.Cars.AddCar(cleanCarName(b.Name), u.ID)
	if err != nil {
		fail(w, 500, err.Error())
		return
	}
	ok(w, map[string]any{"id": id})
}

// manage says whether the person may change this car: a site administrator or its owner.
func (s *Server) manage(r *http.Request, id int64) bool {
	u := userFrom(r)
	return u.Role == "admin" || s.core().Access(u.ID, id) == store.AccessOwner
}

// patchCar: PATCH /api/cars/{id} {name}.
func (s *Server) patchCar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if s.Cars == nil || !s.manage(r, id) {
		fail(w, 403, "only the car's owner or an administrator can do that")
		return
	}
	var b struct{ Name string }
	if decode(r, &b) != nil || cleanCarName(b.Name) == "" {
		fail(w, 400, "give the car a name")
		return
	}
	if err := s.Cars.Rename(id, cleanCarName(b.Name)); err != nil {
		fail(w, 409, err.Error())
		return
	}
	ok(w, "renamed")
}

// removeCar: DELETE /api/cars/{id}. The car stops being followed and disappears from the lists; its database stays on disk.
func (s *Server) removeCar(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if s.Cars == nil || !s.manage(r, id) {
		fail(w, 403, "only the car's owner or an administrator can do that")
		return
	}
	if cars, _ := s.core().ListCars(); len(cars) < 2 {
		fail(w, 409, "the last car cannot be removed")
		return
	}
	if err := s.Cars.Remove(id); err != nil {
		fail(w, 409, err.Error())
		return
	}
	ok(w, "removed")
}
