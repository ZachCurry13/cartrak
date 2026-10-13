package poller

import (
	"time"

	"github.com/ZachCurry13/cartrak/internal/ford"
	"github.com/ZachCurry13/cartrak/internal/trips"
)

// interval picks the next wait: fastest while driving in live mode.
func (p *Poller) interval(st *ford.State) time.Duration {
	switch {
	case trips.Driving(st):
		p.mu.Lock()
		live := time.Now().Before(p.liveUntil)
		p.mu.Unlock()
		if live && p.Iv.Live > 0 {
			if p.Iv.Live < minLive {
				return minLive
			}
			return p.Iv.Live
		}
		return p.Iv.Driving
	case st.ChargeKw != nil && *st.ChargeKw > 0.2:
		return p.Iv.Charging
	case p.PhoneActive() && p.Iv.Driving > 0 && p.Iv.Driving < p.Iv.Parked:
		return p.Iv.Driving // the phone says a trip is starting; the car has not reported it yet
	}
	return p.Iv.Parked
}

// backoff doubles the wait after each failure, capped at 30 minutes.
func backoff(prev, floor time.Duration) time.Duration {
	n := prev * 2
	if n < floor {
		n = floor
	}
	if n > 30*time.Minute {
		n = 30 * time.Minute
	}
	return n
}

// rateLimitWait is how long to wait after the n-th 429 in a row: what Ford asked, at least
// 30 s, and doubling with each repeat (30 s, 60 s, 2 min ... up to 10 min).
func rateLimitWait(asked time.Duration, n int) time.Duration {
	w := minRateLimited
	for i := 1; i < n && w < maxRateLimited; i++ {
		w *= 2
	}
	if asked > w {
		w = asked
	}
	if w > maxRateLimited {
		w = maxRateLimited
	}
	return w
}

// hintFor is how long a phone's "driving" message keeps the poller on the driving interval.
const hintFor = 30 * time.Minute

// PhoneDriving is called when a paired phone says the trip started (on) or ended (off). Starting
// polls Ford now and switches to the driving interval while the car still looks parked;
// ending polls once so the finished trip is closed promptly. Repeats only extend the window.
func (p *Poller) PhoneDriving(on bool) {
	p.mu.Lock()
	was := time.Now().Before(p.hintUntil)
	if on {
		p.hintUntil = time.Now().Add(hintFor)
	} else {
		p.hintUntil = time.Time{}
	}
	p.mu.Unlock()
	if !on || !was {
		p.Now()
	}
}

// PhoneActive reports whether a phone has said a trip is under way recently.
func (p *Poller) PhoneActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Now().Before(p.hintUntil)
}
