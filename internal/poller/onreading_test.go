package poller

import (
	"context"
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestOnReadingSeesEverySavedReadingAndNotFailures(t *testing.T) {
	p := newPoller(t, scripted{state: &ford.State{TS: time.Now(), Ignition: "OFF"}})
	var seen []*ford.State
	p.OnReading = func(s *ford.State) { seen = append(seen, s) }
	p.once(context.Background(), time.Second)
	p.once(context.Background(), time.Second)
	if len(seen) != 2 {
		t.Fatalf("alerts need every reading: %d", len(seen))
	}
	if st := p.Status(); st.LastOK == 0 {
		t.Fatal("the hook runs after the status says the reading succeeded")
	}
	p.P = scripted{err: context.DeadlineExceeded}
	p.once(context.Background(), time.Second)
	if len(seen) != 2 {
		t.Fatal("a failed poll is not a reading")
	}
}
