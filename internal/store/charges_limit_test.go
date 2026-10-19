package store

import "testing"

func TestListChargesReturnsAsManyAsAreAskedFor(t *testing.T) {
	s, _ := Open(t.TempDir())
	t.Cleanup(func() { s.Close() })
	for i := 0; i < 150; i++ {
		c := &Charge{StartTS: int64(1000 + i*100), PeakKw: 7}
		if err := s.CreateCharge(c); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.ListCharges(5000); len(got) != 150 {
		t.Fatalf("a large limit is not a hidden cap of 100: got %d", len(got))
	}
	if got, _ := s.ListCharges(0); len(got) != 100 {
		t.Fatalf("no limit means the default of 100: got %d", len(got))
	}
}
