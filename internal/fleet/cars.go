package fleet

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/ZachCurry13/cartrak/internal/server"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// AddCar makes a new car owned by a person: its folder and database, its poller and the rest, started at once. The
// person is not yet linked to Ford; that is the next step in the app.
func (f *Fleet) AddCar(name string, ownerID int64) (int64, error) {
	if cars, _ := f.Core.ListCars(); len(cars) >= MaxCars {
		return 0, fmt.Errorf("CarTrak follows at most %d cars", MaxCars)
	}
	id, err := f.Core.AddCar(name, "", ownerID)
	if err != nil {
		return 0, err
	}
	dir := "cars/" + strconv.FormatInt(id, 10)
	if err := f.Core.SetCarDir(id, dir); err != nil {
		return 0, err
	}
	if ownerID != 0 {
		_ = f.Core.SetAccess(ownerID, id, store.AccessOwner)
	}
	car, _ := f.Core.GetCar(id)
	if car == nil {
		return 0, errors.New("the car was not saved")
	}
	r, err := f.build(*car, false)
	if err != nil {
		_ = f.Core.ArchiveCar(id)
		return 0, err
	}
	f.mu.Lock()
	f.cars[id] = r
	f.mu.Unlock()
	f.start(f.ctx, r)
	return id, nil
}

// Rename changes a car's name everywhere it is shown.
func (f *Fleet) Rename(id int64, name string) error {
	if err := f.Core.RenameCar(id, name); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r := f.cars[id]; r != nil {
		r.Car.Name = name
		return nil
	}
	return errors.New("no such car")
}

// Remove stops following a car and hides it. Its database is left on disk, so nothing is lost.
func (f *Fleet) Remove(id int64) error {
	if first := f.Front(); first != nil && first.Car.ID == id {
		return errors.New("the first car cannot be removed")
	}
	f.mu.Lock()
	r := f.cars[id]
	delete(f.cars, id)
	f.mu.Unlock()
	if r == nil {
		return errors.New("no such car")
	}
	if r.cancel != nil {
		r.cancel()
	}
	_ = r.S.Close()
	return f.Core.ArchiveCar(id)
}

// Status says whether a car is linked to Ford and when it last answered.
func (f *Fleet) Status(id int64) (linked bool, lastOK int64, found bool) {
	r := f.Runtime(id)
	if r == nil {
		return false, 0, false
	}
	s := r.P.Status()
	return s.Linked, s.LastOK, true
}

// AdminViews says which administrators opened a car without having been given access, and when.
func (f *Fleet) AdminViews(id int64) map[string]int64 {
	r := f.Runtime(id)
	if r == nil {
		return nil
	}
	return server.ReadAdminViews(r.S.GetKV(server.AdminViewsKey))
}

// Inbox is the latest alerts about the cars a person can open, newest first.
func (f *Fleet) Inbox(userID int64, limit int) []server.InboxItem {
	var out []server.InboxItem
	var seen []*Runtime
	for _, r := range f.all() {
		if f.canSee(userID, r.Car.ID) {
			seen = append(seen, r)
		}
	}
	for _, r := range seen {
		xs, _ := r.S.RecentAlerts(limit)
		for _, x := range xs {
			it := server.InboxItem{Car: r.Car.ID, ID: x.ID, TS: x.TS, Kind: x.Kind, Title: x.Title, Body: x.Body, URL: x.URL}
			if len(seen) > 1 {
				it.CarName = r.Car.Name
			}
			out = append(out, it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS > out[j].TS })
	if len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []server.InboxItem{}
	}
	return out
}
