package store

// SetTripNotes writes a trip's notes and leaves its category alone (a calendar appointment's title).
func (s *Store) SetTripNotes(id int64, notes string) error {
	_, err := s.DB.Exec(`UPDATE trips SET notes=? WHERE id=?`, notes, id)
	return err
}
