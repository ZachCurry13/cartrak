package store

import "testing"

func TestCategoriesStartWithFourAndOwnersAddAndRemoveTheirOwn(t *testing.T) {
	s, _ := Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	cs, _ := s.ListCategories()
	if len(cs) != 4 || cs[0].Key != "personal" || cs[1].Key != "work" || cs[2].Key != "medical" || cs[3].Key != "charity" || cs[3].Rate == nil || *cs[3].Rate != 0.14 {
		t.Fatalf("seed: %+v", cs)
	}
	if err := s.DeleteCategory("charity"); err == nil {
		t.Fatal("charity is built in")
	}
	key, err := s.AddCategory("Charity run", "#16a34a", nil)
	if err != nil || key != "charity-run" || !s.CategoryExists(key) {
		t.Fatalf("add: %q %v", key, err)
	}
	if _, err := s.AddCategory("Bad colour", "red", nil); err == nil {
		t.Fatal("a colour must be #rrggbb")
	}
	// two with the same label get different keys
	k2, _ := s.AddCategory("Charity run", "", nil)
	if k2 == key {
		t.Fatal("keys must be unique")
	}
	// a trip, a rule and a mode that use the category go with it
	tr := &Trip{StartTS: 100, Category: key, CategorySource: "manual"}
	_ = s.CreateTrip(tr)
	_, _ = s.DB.Exec(`INSERT INTO rules(name,category) VALUES('r',?)`, key)
	_ = s.AddTimeOff(&TimeOff{From: "2026-01-01", To: "2026-01-02", Kind: key})
	if err := s.DeleteCategory("work"); err == nil {
		t.Fatal("built-in categories cannot be removed")
	}
	if err := s.DeleteCategory(key); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTrip(tr.ID)
	if got.Category != "personal" || s.CategoryExists(key) {
		t.Fatalf("its trips become Personal: %+v", got)
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM rules WHERE category=?`, key).Scan(&n)
	if n != 0 || s.TimeOffKind("2026-01-01") != "" {
		t.Fatal("its rules and modes go too")
	}
}

func TestTheNewestModeWinsAndOnlyRealCategoriesCanBeModes(t *testing.T) {
	s, _ := Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	_ = s.AddTimeOff(&TimeOff{From: "2026-03-01", To: "2026-03-10", Kind: KindVacation})
	_ = s.AddTimeOff(&TimeOff{From: "2026-03-04", To: "2026-03-05", Kind: "work"})
	_ = s.AddTimeOff(&TimeOff{From: "2026-03-06", To: "2026-03-06", Kind: "medical"})
	for day, want := range map[string]string{"2026-03-02": KindVacation, "2026-03-04": "work", "2026-03-06": "medical", "2026-04-01": ""} {
		if got := s.TimeOffKind(day); got != want {
			t.Fatalf("%s: got %q want %q", day, got, want)
		}
	}
	if err := s.AddTimeOff(&TimeOff{From: "2026-05-01", To: "2026-05-02", Kind: "nonsense"}); err == nil {
		t.Fatal("a mode must be a vacation or a real category")
	}
	// a stretch saved before modes had kinds counts as a vacation
	_, _ = s.DB.Exec(`INSERT INTO time_off(from_day,to_day,label) VALUES('2026-06-01','2026-06-02','old')`)
	if s.TimeOffKind("2026-06-01") != KindVacation {
		t.Fatal("older stretches are vacations")
	}
}
