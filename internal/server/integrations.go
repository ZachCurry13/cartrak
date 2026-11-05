package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/afdc"
	"github.com/ZachCurry13/cartrak/internal/alerts"
	"github.com/ZachCurry13/cartrak/internal/gasprice"
	"github.com/ZachCurry13/cartrak/internal/googleplaces"
	"github.com/ZachCurry13/cartrak/internal/routeest"
	"github.com/ZachCurry13/cartrak/internal/settings"
	"github.com/ZachCurry13/cartrak/internal/weather"
)

// integration is one outside service CarTrak can use: what it is for, what it sends, whether it is on, and which settings belong
// to it (the page edits them on the card).
type integration struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Purpose  string      `json:"purpose"`
	Sends    string      `json:"sends"`
	State    string      `json:"state"` // on, off, needs (set up first) or info (managed on its own page)
	Note     string      `json:"note,omitempty"`
	Keys     []string    `json:"keys,omitempty"`
	Link     string      `json:"link,omitempty"`
	LinkText string      `json:"linkText,omitempty"`
	Testable bool        `json:"testable"`
	Last     *testResult `json:"lastTest,omitempty"`
}

type testResult struct {
	TS  int64  `json:"ts"`
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}

const integTestKey = "integ_test:"

func (s *Server) lastTest(id string) *testResult {
	var t testResult
	if json.Unmarshal([]byte(s.S.GetKV(integTestKey+id)), &t) == nil && t.TS > 0 {
		return &t
	}
	return nil
}

func agoText(ts int64) string {
	d := time.Since(time.Unix(ts, 0))
	switch {
	case d < time.Hour:
		return "less than an hour ago"
	case d < 48*time.Hour:
		return strconv.Itoa(int(d.Hours())) + " hours ago"
	}
	return strconv.Itoa(int(d.Hours()/24)) + " days ago"
}

// integrations lists every outside service with its state.
func (s *Server) integrations() []integration {
	c := s.Cfg
	if c == nil {
		return []integration{}
	}
	sites, at, _ := s.knownList(superKey, superAtKey)
	eaOn := c.AFDCKey() != ""
	chargeNote := "Downloaded the first time you plan a long trip, then about weekly."
	if sites > 0 {
		chargeNote = strconv.Itoa(sites) + " sites kept, fetched " + agoText(at.Unix()) + "."
	}
	eaNote := "Needs the free key (developer.nlr.gov/signup)."
	if eaOn {
		eaNote = "Key saved."
		if n, a, _ := s.knownList(eaKey, eaAtKey); n > 0 {
			eaNote = strconv.Itoa(n) + " chargers kept, fetched " + agoText(a.Unix()) + "."
		}
	}
	gasNote := "No key: your own gas price is used."
	if c.EIAKey() != "" {
		gasNote = "Key saved."
		if st := s.gasLiveStatus(); st.OK > 0 {
			gasNote = "Prices last fetched " + agoText(st.OK) + "."
		} else if st.Err != "" {
			gasNote = "The last fetch failed: " + st.Err
		}
	}
	list := []integration{
		{ID: "ford", Name: "Ford (the car)", Purpose: "Where the car's readings, trips and charging come from.", Sends: "Your Ford sign-in only, to Ford. Nothing else leaves.",
			State: map[bool]string{true: "on", false: "needs"}[c.Configured()], Link: "#/ford", LinkText: "Open the Ford connection", Note: map[bool]string{true: "", false: "Not set up yet."}[c.Configured()]},
		{ID: "remote", Name: "Remote access (Cloudflare)", Purpose: "Reaching CarTrak from your phone away from home, and the address links use.", Sends: "Nothing about your trips; it only carries the connection.",
			State: onOff(c.Value(keyAddress) != ""), Link: "#/remote", LinkText: "Open Remote access"},
		{ID: "phone", Name: "Phone GPS", Purpose: "Recording the exact route of a drive from your phone.", Sends: "Your phone's position to this server only.", State: "info", Link: "#/phone", LinkText: "Open Phone tracking"},
		{ID: "maps", Name: "Maps (OpenStreetMap)", Purpose: "The map tiles under your routes.", Sends: "Which map squares to draw, from this server, so the service sees the server and not you.",
			State: onOff(c.MapEnabled()), Keys: []string{settings.MapEnabledKey, settings.MapTileURLKey, settings.MapAttributionKey}},
		{ID: "city", Name: "City names (OpenStreetMap)", Purpose: "Naming where trips start and end, and the address search.", Sends: "A rounded position for a city name; an address only when you search for one.",
			State: onOff(c.GeocodeEnabled()), Keys: []string{settings.GeocodeEnabledKey, settings.GeocodeURLKey}, Testable: true},
		{ID: "road", Name: "Road estimates (routing service)", Purpose: "Real road distance and time for estimated trips and the route planner.", Sends: "The start and end points of a trip being estimated or planned.",
			State: onOff(c.RouteLookup()), Keys: []string{settings.RouteKey, settings.RouteURLKey}, Testable: true},
		{ID: "weather", Name: "Weather (Open-Meteo)", Purpose: "Wind, rain and temperature for trips and the planner.", Sends: "A position and a day, with no account or name.",
			State: onOff(c.WeatherLookup()), Keys: []string{settings.WeatherKey}, Testable: true},
		{ID: "gas", Name: "Gas prices (EIA)", Purpose: "Comparing each charge with the weekly gas price where it happened.", Sends: "The state a charge was in, with your key.",
			State: onOff(c.EIAKey() != ""), Note: gasNote, Keys: []string{settings.EIAKeyKey, settings.GasAdjustKey}, Testable: c.EIAKey() != ""},
		{ID: "super", Name: "Superchargers (supercharge.info)", Purpose: "Which Tesla sites other cars can use, for long trips and for naming charges.", Sends: "Nothing: only the public list is downloaded.",
			State: onOff(sites > 0), Note: chargeNote},
		{ID: "ea", Name: "Electrify America (NLR charger list)", Purpose: "Adds Electrify America chargers to long trips.", Sends: "Your key, to download the public list.",
			State: onOff(eaOn), Note: eaNote, Keys: []string{settings.AFDCKeyKey}, Testable: eaOn},
		{ID: "places", Name: "Google Places (finding companies)", Purpose: "Finding the businesses in a pasted company list the way Google does.", Sends: "A company name and Home's rough area, with your key.",
			State: onOff(c.GooglePlacesKey() != ""), Note: "Without it the map's own search is used, which does not list many small businesses.", Keys: []string{settings.GooglePlacesKeyKey}, Testable: c.GooglePlacesKey() != ""},
		{ID: "uptime", Name: "Uptime monitor", Purpose: "Telling you when CarTrak itself is down, and when it is back. It cannot announce its own outage, so it asks an outside monitor every five minutes; the monitor alerts you when that stops.", Sends: "Only a request to the address you give, every five minutes.",
			State: onOff(c.HeartbeatURL() != ""), Note: "Without it, CarTrak still tells you when it is back after being offline for over five minutes.", Keys: []string{settings.HeartbeatURLKey}, Testable: c.HeartbeatURL() != ""},
		{ID: "push", Name: "Push alerts", Purpose: "Alerts on this phone when something happens.", Sends: "The alert text, through your phone's own push service.", State: "info", Link: "#/notifications", LinkText: "Open Notifications"},
	}
	for i := range list {
		list[i].Last = s.lastTest(list[i].ID)
		if list[i].ID == "uptime" && list[i].State == "on" { // the pings CarTrak makes by itself are its own test
			var h alerts.HeartbeatResult
			if json.Unmarshal([]byte(s.S.GetKV("heartbeat_last")), &h) == nil && h.TS > 0 {
				list[i].Last = &testResult{TS: h.TS, OK: h.OK, Msg: h.Msg}
			}
		}
	}
	return list
}

// knownList is how many sites a kept charger list holds and when it was fetched.
func (s *Server) knownList(key, atKey string) (int, time.Time, bool) {
	var l []json.RawMessage
	if b := s.S.GetKV(key); b == "" || json.Unmarshal([]byte(b), &l) != nil {
		return 0, time.Time{}, false
	}
	n, _ := strconv.ParseInt(s.S.GetKV(atKey), 10, 64)
	return len(l), time.Unix(n, 0), true
}

func (s *Server) gasLiveStatus() gasprice.Status {
	if s.gasLive.svc == nil {
		return gasprice.Status{}
	}
	return s.gasLive.svc.Status()
}

// attention counts the services that need a look: Ford not set up, or a test that last failed on a service that is on.
func attention(l []integration) int {
	n := 0
	for _, x := range l {
		if x.State == "needs" || (x.Last != nil && !x.Last.OK && x.State == "on") {
			n++
		}
	}
	return n
}

// listIntegrations: GET /api/integrations. Admin only.
func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request) {
	l := s.integrations()
	ok(w, map[string]any{"items": l, "attention": attention(l) + s.dataUpdatesAvailable()})
}

// integrationAttention: GET /api/integrations/attention. Just the count, for the badge on the More menu.
func (s *Server) integrationAttention(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]int{"attention": attention(s.integrations()) + s.dataUpdatesAvailable()})
}

// testIntegration: POST /api/integrations/test {"id": "weather"}. Asks the service once, with a short wait, and keeps what happened.
func (s *Server) testIntegration(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID string }
	if decode(r, &in) != nil || s.Cfg == nil {
		fail(w, 400, "say which service")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	var err error
	switch in.ID {
	case "weather":
		c := s.Weather
		if c == nil {
			c = &weather.Client{}
		}
		_, err = c.Day(ctx, 45, -93, time.Now())
	case "road":
		_, _, err = s.routeClient().Drive(ctx, []routeest.Point{{45, -93}, {45.1, -93}})
	case "city":
		if g := s.geoClient(); g == nil {
			fail(w, 400, "City names is off, so there is nothing to test")
			return
		} else {
			_, err = g.City(ctx, 48.8584, 2.2945)
		}
	case "gas":
		if s.Cfg.EIAKey() == "" {
			fail(w, 400, "add the key first")
			return
		}
		_, err = gasprice.Fetch(ctx, nil, s.GasURL, s.Cfg.EIAKey(), gasprice.ForState("MN"), time.Now().AddDate(0, 0, -30))
	case "uptime":
		if s.Cfg.HeartbeatURL() == "" {
			fail(w, 400, "add the address first")
			return
		}
		err = alerts.PingURL(ctx, nil, s.Cfg.HeartbeatURL())
	case "places":
		if s.Cfg.GooglePlacesKey() == "" {
			fail(w, 400, "add the key first")
			return
		}
		err = googleplaces.Ping(ctx, nil, s.PlacesURL, s.Cfg.GooglePlacesKey())
	case "ea":
		if s.Cfg.AFDCKey() == "" {
			fail(w, 400, "add the key first")
			return
		}
		base := s.AFDCURL
		if base == "" {
			base = afdc.DefaultURL
		}
		err = afdc.Ping(ctx, nil, base, s.Cfg.AFDCKey())
	default:
		fail(w, 400, "that service cannot be tested")
		return
	}
	res := testResult{TS: time.Now().Unix(), OK: err == nil, Msg: "It answered."}
	if err != nil {
		res.Msg = strings.ToUpper(err.Error()[:1]) + err.Error()[1:]
	}
	b, _ := json.Marshal(res)
	_ = s.S.SetKV(integTestKey+in.ID, string(b))
	ok(w, res)
}
