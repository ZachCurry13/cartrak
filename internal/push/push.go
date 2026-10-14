// Package push sends Web Push notifications to the phones and computers that asked for them.
// A message is encrypted for one device and handed to that device's push service (Google, Apple
// or Mozilla); only the device can read it. Messages carry short text, never a location.
package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// The alerts a device can ask for.
const (
	ChargeComplete    = "charge_complete"
	PlugIn            = "plug_in"
	PlugOut           = "plug_out"
	LowTire           = "low_tire"
	Low12V            = "low_12v"
	Unlocked          = "unlocked"
	TripStart         = "trip_start"
	TripEnd           = "trip_end"
	FordTrouble       = "ford_trouble"
	PlugReminder      = "plug_reminder"
	MaintenanceDue    = "maintenance_due"
	ArrivePlace       = "arrive_place"
	WeeklySummary     = "weekly_summary"
	BumpWhileParked   = "bump_parked"
	AlarmSounding     = "alarm_sounding"
	LowDisk           = "low_disk"
	LeavePlace        = "leave_place"
	ChargeInterrupted = "charge_interrupted"
)

type Kind struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Help  string `json:"help"`
}

// Kinds is the list the Notifications page offers, in display order.
var Kinds = []Kind{
	{ChargeComplete, "Charging complete", "The battery reached its target or charging stopped."},
	{PlugIn, "Plugged in", "The charging cable was connected."},
	{PlugOut, "Unplugged", "The charging cable was removed."},
	{LowTire, "Low tire pressure", "The car reports a tire as low."},
	{Low12V, "Low 12 V battery", "The small battery is low enough to risk the car not waking. At most once a day."},
	{Unlocked, "Car left unlocked", "Parked and unlocked for more than 10 minutes."},
	{TripStart, "Trip started", "A trip began."},
	{TripEnd, "Trip ended", "A trip finished, with its distance."},
	{ChargeInterrupted, "Charging stopped early", "Charging stopped while still plugged in at a lower level than your sessions usually end at, which can mean a loose cable or a charger fault."},
	{BumpWhileParked, "Bump while parked", "A parked car's motion sensors read above the bump level (More, Settings). The car reports only now and then, so a short knock can be missed."},
	{LowDisk, "Low disk space", "The disk CarTrak stores its data on is getting full. At most one alert every three days."},
	{AlarmSounding, "Car alarm", "The car reports that its alarm or panic alarm is going off."},
	{WeeklySummary, "Weekly summary", "A short summary of the week on Sunday evening: distance, trips, charging and efficiency."},
	{ArrivePlace, "Arrived at a named place", "The car arrived at a place you named under More, Places, such as Home or the office."},
	{LeavePlace, "Left a named place", "The car left a place you named under More, Places."},
	{MaintenanceDue, "Service due", "A service reminder you set up under More, Maintenance has come due by distance or by time."},
	{PlugReminder, "Plug in tonight?", "One reminder in the evening if the car is parked, not plugged in and the battery is low. An admin sets the time and level."},
	{FordTrouble, "Ford connection trouble", "No reading from the car for several hours."},
}

func ValidKind(id string) bool {
	for _, k := range Kinds {
		if k.ID == id {
			return true
		}
	}
	return false
}

// Alert is one notification. URL is the page to open when it is tapped.
type Alert struct {
	Kind  string
	Title string
	Body  string
	URL   string
}

// KV stores the VAPID keys (the server's identity with the push services).
type KV interface {
	GetKV(k string) string
	SetKV(k, v string) error
}

// Subs is the part of the store the sender needs.
type Subs interface {
	ListPushSubs(userID int64) ([]*store.PushSub, error)
	MarkPush(id int64, errText string) error
	DeletePushSubID(id int64) error
}

type Service struct {
	KV       KV
	Subs     Subs
	HTTP     *http.Client
	Location func() *time.Location // for quiet hours
	AllowAny bool                  // tests only: accept any push address
	// Log, when set, is told about every alert: how many devices wanted it, were in quiet hours, and accepted it.
	Log func(a Alert, devices, wanted, quiet, sent int)
}

const (
	kvPrivate = "vapid_private"
	kvPublic  = "vapid_public"
	subject   = "https://github.com/ZachCurry13/cartrak"
)

// PublicKey is the server's public VAPID key, made on first use. The private half never leaves the server.
func (s *Service) PublicKey() (string, error) {
	_, pub, err := s.keys()
	return pub, err
}

func (s *Service) keys() (priv, pub string, err error) {
	if priv, pub = s.KV.GetKV(kvPrivate), s.KV.GetKV(kvPublic); priv != "" && pub != "" {
		return priv, pub, nil
	}
	if priv, pub, err = webpush.GenerateVAPIDKeys(); err != nil {
		return "", "", err
	}
	if err = s.KV.SetKV(kvPrivate, priv); err == nil {
		err = s.KV.SetKV(kvPublic, pub)
	}
	return priv, pub, err
}

// allowedHosts are the push services' domains. A subscription's address is chosen by the browser,
// but a signed-in person could send any address, so only these are ever contacted.
var allowedHosts = []string{"googleapis.com", "push.services.mozilla.com", "push.apple.com", "notify.windows.com"}

// AllowedEndpoint says whether CarTrak will deliver to this push address.
func AllowedEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, a := range allowedHosts {
		if h == a || strings.HasSuffix(h, "."+a) {
			return true
		}
	}
	return false
}

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
	for _, sub := range subs {
		if !has(sub.Events, a.Kind) {
			continue
		}
		if InQuiet(sub.Quiet, now) {
			quiet++
			continue
		}
		wanted++
		if s.SendTo(ctx, sub, a) == nil {
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
