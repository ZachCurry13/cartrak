package fleet

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BackupAll writes a zip of the whole installation: the core database (the people, without their sign-in sessions),
// every car's database (without accounts), and a list saying which file is which car. tmpDir is where each copy is made
// before it is added.
func (f *Fleet) BackupAll(tmpDir string, w io.Writer) error {
	zw := zip.NewWriter(w)
	add := func(name, path string) error {
		defer os.Remove(path)
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		return err
	}
	stamp := fmt.Sprint(time.Now().UnixNano())
	corePath := filepath.Join(tmpDir, "site-core-"+stamp+".tmp")
	if err := f.Core.Backup(corePath, false); err != nil {
		return err
	}
	if err := add("core.db", corePath); err != nil {
		return err
	}
	type entry struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		File string `json:"file"`
	}
	var list []entry
	runtimes := f.all()
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].Car.ID < runtimes[j].Car.ID })
	for _, r := range runtimes {
		p := filepath.Join(tmpDir, fmt.Sprintf("site-car%d-%s.tmp", r.Car.ID, stamp))
		if err := r.S.BackupCar(p, false); err != nil {
			return err
		}
		name := fmt.Sprintf("car-%d.db", r.Car.ID)
		if err := add(name, p); err != nil {
			return err
		}
		list = append(list, entry{r.Car.ID, r.Car.Name, name})
	}
	m, err := zw.Create("cars.json")
	if err != nil {
		return err
	}
	if err := json.NewEncoder(m).Encode(list); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	for _, r := range runtimes { // every car counts as backed up, for the backup reminder
		_ = r.S.SetKV("backup_taken_at", fmt.Sprint(time.Now().Unix()))
	}
	return nil
}
