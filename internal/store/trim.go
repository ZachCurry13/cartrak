package store

import (
	"context"
	"time"
)

// StorageCounts says how many readings are older than cutoff (unix seconds): the stored raw Ford copies
// (snapshots) and the readings' full detail (samples). Only the time index is used, so it is fast.
func (s *Store) StorageCounts(cutoff int64) (snapshots, samples int, err error) {
	if err = s.DB.QueryRow(`SELECT COUNT(*) FROM snapshots WHERE ts<?`, cutoff).Scan(&snapshots); err != nil {
		return
	}
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM samples WHERE ts<?`, cutoff).Scan(&samples)
	return
}

// TrimRaw removes the raw Ford copy of every reading older than cutoff and empties each such reading's full
// detail. The values CarTrak charts and totals are kept (trips, charging, places, trends, positions, the main
// numbers of every reading). It works in small batches with a pause between, so the app stays responsive,
// and reports progress. Disk space freed this way is reused for new readings, so the database stops growing
// for a while; a full backup restored afterwards is compact.
func (s *Store) TrimRaw(ctx context.Context, cutoff int64, batch int, pause time.Duration, progress func(done int)) error {
	done := 0
	for _, q := range []string{
		`DELETE FROM snapshots WHERE id IN (SELECT id FROM snapshots WHERE ts<? LIMIT ?)`,
		`UPDATE samples SET extra='{}' WHERE id IN (SELECT id FROM samples WHERE ts<? AND extra IS NOT NULL AND extra!='{}' LIMIT ?)`,
	} {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			res, err := s.DB.Exec(q, cutoff, batch)
			if err != nil {
				return err
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				break
			}
			done += int(n)
			if progress != nil {
				progress(done)
			}
			time.Sleep(pause)
		}
	}
	return nil
}
