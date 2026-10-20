package poller

import "time"

// PollRec is one poll: when, how it went, and how long CarTrak decided to wait before the next one.
type PollRec struct {
	TS      int64  `json:"ts"`
	Result  string `json:"result"` // ok, rate limit, Ford error or failed
	NextSec int    `json:"nextSec"`
	Detail  string `json:"detail"`
}

const keepPolls = 400

func (p *Poller) record(r PollRec) {
	if len(r.Detail) > 160 {
		r.Detail = r.Detail[:160]
	}
	p.mu.Lock()
	p.polls = append(p.polls, r)
	if len(p.polls) > keepPolls {
		p.polls = append([]PollRec(nil), p.polls[len(p.polls)-keepPolls:]...)
	}
	p.mu.Unlock()
}

// RecentPolls is the polls since a time, oldest first.
func (p *Poller) RecentPolls(since time.Time) []PollRec {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []PollRec
	for _, r := range p.polls {
		if r.TS >= since.Unix() {
			out = append(out, r)
		}
	}
	return out
}
