package store

import (
	"encoding/json"
	"fmt"
)

// A person's own choices that follow them to every device: their birthday (month and day, never the year) and whether
// friendly alerts get a holiday touch. Only that person reads or changes them.

// UserPrefs is one person's choices.
type UserPrefs struct {
	Birthday     string `json:"birthday"`     // "MM-DD", or empty
	HolidayOff   bool   `json:"holidayOff"`   // no holiday or birthday touches on alerts, and no birthday alert
	HolidayVoice bool   `json:"holidayVoice"` // friendly alerts also talk in the holiday's voice (Santa, a friendly ghost, a leprechaun)
}

func prefsKey(uid int64) string { return fmt.Sprintf("prefs:%d", uid) }

// Prefs is a person's choices; nothing chosen is the zero value (holiday touches on, no birthday).
func (s *Store) Prefs(uid int64) UserPrefs {
	var p UserPrefs
	if raw := s.GetKV(prefsKey(uid)); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p)
	}
	return p
}

func (s *Store) SetPrefs(uid int64, p UserPrefs) error {
	b, _ := json.Marshal(p)
	return s.SetKV(prefsKey(uid), string(b))
}

// forgetPrefs removes a person's choices, when the person is removed.
func (s *Store) forgetPrefs(uid int64) {
	_, _ = s.DB.Exec(`DELETE FROM kv WHERE k=? OR k=?`, prefsKey(uid), fmt.Sprintf("layout:%d", uid))
}
