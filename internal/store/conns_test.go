package store

import (
	"fmt"
	"sync"
	"testing"
)

// Writers and readers at the same time must never fail with "database is locked", and every write must land.
func TestReadersAndWritersAtOnceNeverCollide(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.DB.Exec(`CREATE TABLE conc (id INTEGER PRIMARY KEY, who INTEGER, n INTEGER)`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 150; i++ {
				if i%3 == 0 { // a transaction that reads and then writes: the case that fails when a writer gets in between
					tx, err := s.DB.Begin()
					if err != nil {
						errs <- err
						return
					}
					var c int
					if err := tx.QueryRow(`SELECT COUNT(*) FROM conc`).Scan(&c); err != nil {
						errs <- err
						return
					}
					if _, err := tx.Exec(`INSERT INTO conc(who,n) VALUES(?,?)`, w, c); err != nil {
						errs <- fmt.Errorf("tx insert: %w", err)
						return
					}
					if err := tx.Commit(); err != nil {
						errs <- err
						return
					}
				} else if _, err := s.DB.Exec(`INSERT INTO conc(who,n) VALUES(?,?)`, w, i); err != nil {
					errs <- fmt.Errorf("insert: %w", err)
					return
				}
			}
		}(w)
	}
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				var c int
				if err := s.DB.QueryRow(`SELECT COUNT(*) FROM conc`).Scan(&c); err != nil {
					errs <- fmt.Errorf("read: %w", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM conc`).Scan(&total); err != nil || total != 600 {
		t.Fatalf("every write landed: %d %v", total, err)
	}
}
