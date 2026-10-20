package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestImportIdentityKeepsPeopleAndIds(t *testing.T) {
	dir := t.TempDir()
	car, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer car.Close()
	if err := car.CreateUser("ann", "h1", "admin"); err != nil {
		t.Fatal(err)
	}
	if err := car.CreateUser("bob", "h2", "viewer"); err != nil {
		t.Fatal(err)
	}
	bob, _ := car.UserByName("bob")
	if err := car.NewSession("tok", bob.ID, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := car.SavePushSub(&PushSub{UserID: bob.ID, Endpoint: "https://fcm.googleapis.com/x", P256dh: "p", Auth: "a", Events: []string{"trip_end"}}); err != nil {
		t.Fatal(err)
	}
	_ = car.SetKV("vapid_private", "secret")
	_ = car.SetKV("not_global", "x")

	core, err := OpenCore(filepath.Join(dir, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	if err := ImportIdentity(core, car, []string{"vapid_private"}); err != nil {
		t.Fatal(err)
	}
	u, _ := core.SessionUser("tok")
	if u == nil || u.Name != "bob" || u.ID != bob.ID {
		t.Fatalf("session user not carried over: %+v", u)
	}
	if subs, _ := core.ListPushSubs(bob.ID); len(subs) != 1 {
		t.Fatalf("push subscriptions: %d", len(subs))
	}
	if core.GetKV("vapid_private") != "secret" || core.GetKV("not_global") != "" {
		t.Fatal("only the listed keys should be copied")
	}
	// a second import does nothing
	if err := ImportIdentity(core, car, nil); err != nil || core.CountUsers() != 2 {
		t.Fatalf("second import: %v, %d users", err, core.CountUsers())
	}
	if car.CountUsers() != 2 {
		t.Fatal("the car's own copy must be left alone")
	}
}

func TestCarsAndAccess(t *testing.T) {
	core, err := OpenCore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	_ = core.CreateUser("ann", "h", "admin")
	_ = core.CreateUser("bob", "h", "viewer")
	bob, _ := core.UserByName("bob")
	first, _ := core.AddCar("Mach-E", "", 0)
	second, _ := core.AddCar("Bob's truck", "cars/2", bob.ID)
	if first != 1 || second != 2 {
		t.Fatalf("ids %d %d", first, second)
	}
	_ = core.SetAccess(bob.ID, second, AccessOwner)
	_ = core.SetAccess(bob.ID, first, AccessViewer)
	if core.Access(bob.ID, second) != AccessOwner || core.Access(bob.ID, first) != AccessViewer {
		t.Fatal("access not stored")
	}
	_ = core.SetAccess(bob.ID, first, "")
	if core.Access(bob.ID, first) != "" {
		t.Fatal("access not removed")
	}
	_ = core.SetOwnCar(bob.ID, true)
	if !core.OwnCar(bob.ID) {
		t.Fatal("own-car flag")
	}
	if us, _ := core.ListUsers(); !us[1].OwnCar || us[0].OwnCar {
		t.Fatalf("list users flags: %+v", us)
	}
	_ = core.ArchiveCar(second)
	if cs, _ := core.ListCars(); len(cs) != 1 || cs[0].Name != "Mach-E" {
		t.Fatalf("archived car still listed: %+v", cs)
	}
	_ = core.DeleteUser(bob.ID)
	if m, _ := core.UserAccess(bob.ID); len(m) != 0 {
		t.Fatal("access should go with the person")
	}
}
