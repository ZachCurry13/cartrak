package poller

import (
	"testing"
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
)

func TestPhoneSaysDrivingSpeedsUpWhileTheCarLooksParked(t *testing.T) {
	p := newPoller(t, scripted{})
	parked := &ford.State{Ignition: "OFF"}

	if got := p.interval(parked); got != 10*time.Minute {
		t.Fatalf("parked before the phone speaks: %v", got)
	}
	p.PhoneDriving(true)
	if got := p.interval(parked); got != 90*time.Second {
		t.Fatalf("parked but the phone says driving: %v", got)
	}
	select {
	case <-p.kick:
	default:
		t.Fatal("a trip start should ask for an immediate reading")
	}
	p.PhoneDriving(true) // another ping only extends the window
	select {
	case <-p.kick:
		t.Fatal("a repeat message must not trigger another reading")
	default:
	}
	p.PhoneDriving(false)
	if got := p.interval(parked); got != 10*time.Minute {
		t.Fatalf("trip ended, back to the parked pace: %v", got)
	}
	select {
	case <-p.kick:
	default:
		t.Fatal("a trip end should ask for a reading so the trip closes")
	}
}

func TestPhoneHintExpires(t *testing.T) {
	p := newPoller(t, scripted{})
	p.PhoneDriving(true)
	p.mu.Lock()
	p.hintUntil = time.Now().Add(-time.Second)
	p.mu.Unlock()
	if got := p.interval(&ford.State{Ignition: "OFF"}); got != 10*time.Minute {
		t.Fatalf("an old hint must not keep polling fast: %v", got)
	}
}
