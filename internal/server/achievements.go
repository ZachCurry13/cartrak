package server

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/analytics"
	"github.com/ZachCurry13/cartrak/internal/push"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Once an hour CarTrak looks for badges earned since it last looked, and on the first of the month says last month's Wrapped is
// ready. Both go to the Inbox, and to phones that turned those alerts on.

const (
	kvBadgesKnown = "badges_known"
	kvRecapSent   = "recap_sent"
)

func (s *Server) noticeLoop(ctx context.Context) {
	for {
		s.noticeOnce(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
		}
	}
}

func (s *Server) noticeOnce(ctx context.Context, now time.Time) {
	s.newBadges(ctx)
	s.recapReady(ctx, now)
}

func (s *Server) tell(ctx context.Context, a push.Alert) {
	if s.Push != nil {
		s.Push.Notify(ctx, a)
		return
	}
	_ = s.S.LogAlert(store.AlertLogEntry{TS: time.Now().Unix(), Kind: a.Kind, Title: a.Title, Body: a.Body, URL: a.URL})
}

// newBadges tells about badges earned since the last look. The first look only notes what is already earned: a long history
// should not arrive as a burst of old news.
func (s *Server) newBadges(ctx context.Context) {
	ts, raw, err1 := s.tripsWithPlaces(0)
	cs, err2 := s.analyticsCharges(0)
	if err1 != nil || err2 != nil {
		return
	}
	known := map[string]bool{}
	prev := s.S.GetKV(kvBadgesKnown)
	for _, k := range strings.Split(prev, ",") {
		known[k] = true
	}
	var have []string
	for _, b := range s.allBadges(ts, cs, s.towns(raw), s.defaultLoc()) {
		if b.Earned == 0 {
			continue
		}
		have = append(have, b.Key)
		if prev != "" && !known[b.Key] {
			s.tell(ctx, push.Alert{Kind: push.Achievement, Title: "Achievement: " + b.Name + " " + b.Emoji, Body: b.About + ".", URL: "#/analytics"})
		}
	}
	sort.Strings(have)
	_ = s.S.SetKV(kvBadgesKnown, "started,"+strings.Join(have, ","))
}

// recapReady says on the first of the month, from 9 am, that last month's Wrapped is ready, when there was driving in it.
func (s *Server) recapReady(ctx context.Context, now time.Time) {
	loc := s.defaultLoc()
	local := now.In(loc)
	if local.Day() != 1 || local.Hour() < 9 {
		return
	}
	month := local.AddDate(0, -1, 0).Format("2006-01")
	if s.S.GetKV(kvRecapSent) == month {
		return
	}
	_ = s.S.SetKV(kvRecapSent, month)
	from, to, _, _ := analytics.Period(month, loc)
	if ts, err := s.S.ListTrips(store.TripFilter{From: from, To: to, Limit: 1}); err != nil || len(ts) == 0 {
		return
	}
	name := local.AddDate(0, -1, 0).Format("January")
	s.tell(ctx, push.Alert{Kind: push.MonthlyRecap, Title: "Your " + name + ", wrapped ✨", Body: "Miles, energy, savings, your car's personality and more.", URL: "#/wrapped/" + month})
}
