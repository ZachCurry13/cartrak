package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// CoreFile is the database of the people, sign-ins, notification devices and the list of cars. The cars' own readings,
// trips and charging stay in their own databases, so one car's data can never leak into another's.
const CoreFile = "core.db"

// OpenCore opens (or creates) the core database in the data folder.
func OpenCore(dir string) (*Store, error) { return openFile(dir, CoreFile) }

// CoreExists says whether the core database has been made yet.
func CoreExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, CoreFile))
	return err == nil
}

// ImportIdentity copies the people, their sign-ins, notification devices and the few settings that belong to the whole
// installation from a car's database into the core database, keeping every id so sessions and devices stay valid. The
// car's own copies are left where they are. It does nothing when the core already has people.
func ImportIdentity(core, car *Store, globalKeys []string) error {
	if core.CountUsers() > 0 {
		return nil
	}
	// The core database has one connection, so the attach stays with the copy that follows. ATTACH cannot run inside
	// a transaction, hence before it.
	if _, err := core.DB.Exec(`ATTACH DATABASE ? AS src`, "file:"+filepath.ToSlash(car.Path)+"?mode=ro"); err != nil {
		return fmt.Errorf("attach: %w", err)
	}
	defer func() { _, _ = core.DB.Exec(`DETACH DATABASE src`) }()
	tx, err := core.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`INSERT INTO users(id,name,hash,role) SELECT id,name,hash,role FROM src.users`,
		`INSERT INTO sessions(token,user_id,expires,created,ip,agent) SELECT token,user_id,expires,created,ip,agent FROM src.sessions`,
		`INSERT INTO signins(id,ts,name,ok,ip,agent) SELECT id,ts,name,ok,ip,agent FROM src.signins`,
		`INSERT INTO push_subs(id,user_id,endpoint,p256dh,auth,label,events,quiet,created,last_ok,last_err)
			SELECT id,user_id,endpoint,p256dh,auth,label,events,quiet,created,last_ok,last_err FROM src.push_subs`,
	} {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("import: %w", err)
		}
	}
	for _, k := range globalKeys {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO kv(k,v) SELECT k,v FROM src.kv WHERE k=?`, k); err != nil {
			return err
		}
	}
	return tx.Commit()
}
