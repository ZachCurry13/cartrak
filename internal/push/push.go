// Package push sends Web Push notifications to the phones and computers that asked for them.
// A message is encrypted for one device and handed to that device's push service (Google, Apple
// or Mozilla); only the device can read it. Messages carry short text, never a location.
package push

import (
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
	LowBattery        = "low_battery"
	ChargeBudget      = "charge_budget"
	BackupReminder    = "backup_reminder"
	LeftOpen          = "left_open"
	CarFault          = "car_fault"
	ChargeSlow        = "charge_slow"
	Achievement       = "achievement"
	MonthlyRecap      = "monthly_recap"
	ServiceVisit      = "service_visit"
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
	{LowBattery, "Low battery", "The battery fell to the car's Battery reserve (More, Charging settings; 20% unless chosen) while the car was not charging."},
	{ChargeBudget, "Charging budget", "This month's charging cost passed the budget an admin set under More, Settings. At most once a month."},
	{LeftOpen, "Left open", "The parked car has a door, the tailgate, the hood or a window open for ten minutes."},
	{BackupReminder, "Backup reminder", "No backup has been downloaded in the last month. At most once a month, and only after CarTrak has been recording for two weeks."},
	{LowTire, "Low tire pressure", "The car reports a tire as low."},
	{CarFault, "Car-reported problem", "The car's own checks report something other than normal: the main battery, the tire pressure system or the 12 V load. At most once a day for each."},
	{ChargeSlow, "Slow fast charge", "A fast charge that has just ended averaged much less than your fast charges usually do, so a different charger or stall may be better. Only charges that stopped by about 85% are judged, because every charger slows down near full. Needs a few earlier fast sessions."},
	{Low12V, "Low 12 V battery", "The small battery is low enough to risk the car not waking. At most once a day."},
	{Unlocked, "Car left unlocked", "Parked and unlocked for more than 10 minutes."},
	{TripStart, "Trip started", "A trip began."},
	{TripEnd, "Trip ended", "A trip finished, with its distance."},
	{ChargeInterrupted, "Charging stopped early", "Charging stopped while still plugged in at a lower level than your sessions usually end at, which can mean a loose cable or a charger fault."},
	{BumpWhileParked, "Bump while parked", "A parked car's motion sensors read above the bump level (More, Settings). The car reports only now and then, so a short knock can be missed."},
	{LowDisk, "Low disk space", "The disk CarTrak stores its data on is getting full. At most one alert every three days."},
	{AlarmSounding, "Car alarm", "The car reports that its alarm or panic alarm is going off."},
	{ServiceVisit, "Service visit", "After the car stayed at a dealership, tire shop or repair shop you marked under Places: was it serviced? Log what was done and what it cost."},
	{Achievement, "Achievement earned", "You earned a new badge in Insights, such as a 1,000-mile month or the 150 kW club."},
	{MonthlyRecap, "Your month, wrapped", "On the first of the month, when last month's Wrapped is ready to look at."},
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
	// Visible, when set, says whether a person may be told about this car (several cars share one list of devices).
	Visible func(userID int64) bool
	// CarName, when set and not empty, is put before the title so people with several cars know which one it is about.
	CarName func() string
	// Prefs, when set, is each person's birthday and whether they want holiday touches on friendly alerts.
	Prefs func(userID int64) store.UserPrefs
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
