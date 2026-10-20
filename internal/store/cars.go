package store

import (
	"database/sql"
	"errors"
	"time"
)

// Car is one vehicle CarTrak follows. Each car has its own database; this list lives in the core database that also
// holds the people. Dir is empty for the first car, whose database sits in the data folder itself.
type Car struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Dir      string `json:"-"`
	Created  int64  `json:"created"`
	OwnerID  int64  `json:"ownerId"` // the person who brought the car; 0 for the first car
	Archived bool   `json:"archived"`
}

// Levels of access a person can have to a car.
const (
	AccessOwner  = "owner"  // manages the car: links Ford, changes its settings
	AccessViewer = "viewer" // reads it
)

const schemaCars = `
CREATE TABLE IF NOT EXISTS cars (
  id INTEGER PRIMARY KEY, name TEXT NOT NULL, dir TEXT NOT NULL DEFAULT '', created INTEGER NOT NULL,
  owner_id INTEGER NOT NULL DEFAULT 0, archived INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS car_access (
  user_id INTEGER NOT NULL, car_id INTEGER NOT NULL, level TEXT NOT NULL, PRIMARY KEY(user_id, car_id));
`

const carCols = `id,name,dir,created,owner_id,archived`

func scanCar(r interface{ Scan(...any) error }) (*Car, error) {
	var c Car
	var arch int
	if err := r.Scan(&c.ID, &c.Name, &c.Dir, &c.Created, &c.OwnerID, &arch); err != nil {
		return nil, err
	}
	c.Archived = arch != 0
	return &c, nil
}

// ListCars returns every car that is not archived, oldest first.
func (s *Store) ListCars() ([]Car, error) {
	rows, err := s.DB.Query(`SELECT ` + carCols + ` FROM cars WHERE archived=0 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Car{}
	for rows.Next() {
		c, err := scanCar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// GetCar returns one car, or nil when there is none with that id.
func (s *Store) GetCar(id int64) (*Car, error) {
	c, err := scanCar(s.DB.QueryRow(`SELECT `+carCols+` FROM cars WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// AddCar records a new car and returns its id. The first car (dir "") is the one that already existed.
func (s *Store) AddCar(name, dir string, ownerID int64) (int64, error) {
	r, err := s.DB.Exec(`INSERT INTO cars(name,dir,created,owner_id) VALUES(?,?,?,?)`, name, dir, time.Now().Unix(), ownerID)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// SetCarDir sets the folder of a car's database, once its id is known.
func (s *Store) SetCarDir(id int64, dir string) error {
	_, err := s.DB.Exec(`UPDATE cars SET dir=? WHERE id=?`, dir, id)
	return err
}

func (s *Store) RenameCar(id int64, name string) error {
	_, err := s.DB.Exec(`UPDATE cars SET name=? WHERE id=?`, name, id)
	return err
}

// ArchiveCar hides a car and stops following it. Its database file is left where it is.
func (s *Store) ArchiveCar(id int64) error {
	_, err := s.DB.Exec(`UPDATE cars SET archived=1 WHERE id=?`, id)
	return err
}

// SetAccess gives a person a level of access to a car; an empty level takes it away.
func (s *Store) SetAccess(userID, carID int64, level string) error {
	if level == "" {
		_, err := s.DB.Exec(`DELETE FROM car_access WHERE user_id=? AND car_id=?`, userID, carID)
		return err
	}
	_, err := s.DB.Exec(`INSERT INTO car_access(user_id,car_id,level) VALUES(?,?,?)
		ON CONFLICT(user_id,car_id) DO UPDATE SET level=excluded.level`, userID, carID, level)
	return err
}

// Access is the level a person has to a car, or "" for none. It does not consider the admin role.
func (s *Store) Access(userID, carID int64) string {
	var l string
	_ = s.DB.QueryRow(`SELECT level FROM car_access WHERE user_id=? AND car_id=?`, userID, carID).Scan(&l)
	return l
}

// CarAccess lists the people who have access to a car, with their level.
func (s *Store) CarAccess(carID int64) (map[int64]string, error) {
	rows, err := s.DB.Query(`SELECT user_id,level FROM car_access WHERE car_id=?`, carID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var u int64
		var l string
		if err := rows.Scan(&u, &l); err != nil {
			return nil, err
		}
		out[u] = l
	}
	return out, rows.Err()
}

// UserAccess lists the cars a person has access to (not counting the admin role), by car id.
func (s *Store) UserAccess(userID int64) (map[int64]string, error) {
	rows, err := s.DB.Query(`SELECT car_id,level FROM car_access WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var c int64
		var l string
		if err := rows.Scan(&c, &l); err != nil {
			return nil, err
		}
		out[c] = l
	}
	return out, rows.Err()
}

// SetOwnCar says whether a person may bring a car of their own.
func (s *Store) SetOwnCar(userID int64, on bool) error {
	_, err := s.DB.Exec(`UPDATE users SET own_car=? WHERE id=?`, b2i(on), userID)
	return err
}

// OwnCar says whether a person may bring a car of their own.
func (s *Store) OwnCar(userID int64) bool {
	var n int
	_ = s.DB.QueryRow(`SELECT own_car FROM users WHERE id=?`, userID).Scan(&n)
	return n != 0
}
