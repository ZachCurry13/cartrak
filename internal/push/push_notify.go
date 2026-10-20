package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/ZachCurry13/cartrak/internal/store"
)

// Notify sends the alert to every device that wants it and is not in quiet hours.
func (s *Service) Notify(ctx context.Context, a Alert) {
	subs, err := s.Subs.ListPushSubs(0)
	if err != nil {
		return
	}
	now := time.Now()
	if s.Location != nil {
		now = now.In(s.Location())
	}
	wanted, quiet, sent := 0, 0, 0
	out := a
	if s.CarName != nil {
		if n := s.CarName(); n != "" {
			out.Title = n + ": " + a.Title
		}
	}
	for _, sub := range subs {
		if s.Visible != nil && !s.Visible(sub.UserID) {
			continue
		}
		if !has(sub.Events, a.Kind) {
			continue
		}
		if InQuiet(sub.Quiet, now) {
			quiet++
			continue
		}
		wanted++
		if s.SendTo(ctx, sub, out) == nil {
			sent++
		}
	}
	if s.Log != nil {
		s.Log(a, len(subs), wanted, quiet, sent)
	}
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// SendTo delivers one alert to one device and records how it went. A device the push service says
// is gone (HTTP 404 or 410) is removed.
func (s *Service) SendTo(ctx context.Context, sub *store.PushSub, a Alert) error {
	if !s.AllowAny && !AllowedEndpoint(sub.Endpoint) {
		return fmt.Errorf("not a known push service address")
	}
	priv, pub, err := s.keys()
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"title": a.Title, "body": a.Body, "tag": a.Kind, "url": a.URL})
	hc := s.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := webpush.SendNotificationWithContext(ctx, body,
		&webpush.Subscription{Endpoint: sub.Endpoint, Keys: webpush.Keys{Auth: sub.Auth, P256dh: sub.P256dh}},
		&webpush.Options{HTTPClient: hc, Subscriber: subject, VAPIDPublicKey: pub, VAPIDPrivateKey: priv, TTL: 3600, Urgency: webpush.UrgencyNormal})
	if err != nil {
		_ = s.Subs.MarkPush(sub.ID, err.Error())
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		_ = s.Subs.DeletePushSubID(sub.ID)
		return fmt.Errorf("the device no longer accepts notifications (HTTP %d); removed", resp.StatusCode)
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		_ = s.Subs.MarkPush(sub.ID, "")
		return nil
	}
	err = fmt.Errorf("the push service answered HTTP %d", resp.StatusCode)
	_ = s.Subs.MarkPush(sub.ID, err.Error())
	return err
}

// InQuiet reports whether the time falls in quiet hours like "22:00-07:00" (which wrap midnight).
func InQuiet(quiet string, now time.Time) bool {
	a, b, ok := strings.Cut(quiet, "-")
	if !ok {
		return false
	}
	from, ok1 := minutes(a)
	to, ok2 := minutes(b)
	if !ok1 || !ok2 || from == to {
		return false
	}
	m := now.Hour()*60 + now.Minute()
	if from < to {
		return m >= from && m < to
	}
	return m >= from || m < to
}

// ValidQuiet checks a quiet hours value; empty means none.
func ValidQuiet(q string) bool {
	if q == "" {
		return true
	}
	a, b, ok := strings.Cut(q, "-")
	_, ok1 := minutes(a)
	_, ok2 := minutes(b)
	return ok && ok1 && ok2
}

func minutes(s string) (int, bool) {
	var h, m int
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	if _, err := fmt.Sscanf(s, "%02d:%02d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
