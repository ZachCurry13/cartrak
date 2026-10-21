package ford

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// FordConnect 2.0: bearer-token queries under /fcon-query/v1 and an Azure B2C
// sign-in. Addresses come from Ford's FordConnect 2.0 Postman collection. The
// sign-in (authorize) address is not in that collection: it follows the same
// pattern as the token address, and can be overridden (FORD_AUTH_URL).
const (
	v2Host     = "https://api.vehicle.ford.com"
	v2TokenURL = v2Host + "/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token?p=B2C_1A_FCON_AUTHORIZE"
	// Ford's own sign-in and vehicle-consent pages start here (what other FordConnect clients use).
	v2LinkURL = v2Host + "/fcon-public/v1/auth/init"
	// The back-end policy answers a redirect address that is not registered with an error, so it is
	// used only to check which addresses Ford accepts, never to sign anyone in.
	v2CheckURL  = v2Host + "/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/authorize?p=B2C_1A_FCON_AUTHORIZE"
	v2QueryBase = v2Host + "/fcon-query/v1"
	slowEvery   = 30 * time.Minute // how often the rarely-changing endpoints are re-read
	refreshKey  = "ford2_refresh"
)

// slowEndpoints change rarely, so they are read at most every slowEvery.
var slowEndpoints = []struct{ Key, Path string }{
	{"garage", "/garage"},
	{"alerts", "/vehicle-health/alerts"},
	{"wallbox", "/wallbox"},
	{"departureTimes", "/electric/departure-times"},
	{"chargeSchedules", "/electric/charge-schedules"},
}

type Connect2Opts struct {
	ClientID, ClientSecret, RedirectURL, AuthURL string
	VIN                                          string          // the vehicle this car follows when the account authorized several; "" lets it choose
	Taken                                        func() []string // vehicles other cars follow, so one left unchosen takes another
	CheckURL                                     string          // address used to test redirect addresses (tests override it)
	APIBase, TokenURL                            string          // overridden in tests
}

type Connect2 struct {
	optsFn func() Connect2Opts // read on every call so admin-panel changes apply live
	store  TokenStore
	hc     *http.Client

	mu      sync.Mutex
	acc     string
	exp     time.Time
	slow    map[string]json.RawMessage
	slowErr map[string]string
	slowAt  map[string]time.Time // when each slow endpoint was last read

	cache     map[string]cachedAnswer // recent good answers, served instead of asking Ford again too soon
	holdUntil time.Time               // Ford asked us to wait: nothing is sent before this
	minGap    time.Duration           // tests shorten it; zero means the default
	vehicles  []VehicleChoice         // every vehicle the linked account authorized, from the latest answer
}

func NewConnect2(fn func() Connect2Opts, st TokenStore) *Connect2 {
	return &Connect2{optsFn: fn, store: st, hc: &http.Client{Timeout: 30 * time.Second},
		slow: map[string]json.RawMessage{}, slowErr: map[string]string{}, slowAt: map[string]time.Time{},
		cache: map[string]cachedAnswer{}}
}

func (c *Connect2) opts() Connect2Opts {
	o := c.optsFn()
	if o.APIBase == "" {
		o.APIBase = v2QueryBase
	}
	if o.TokenURL == "" {
		o.TokenURL = v2TokenURL
	}
	if o.AuthURL == "" {
		o.AuthURL = v2LinkURL
	}
	if o.CheckURL == "" {
		o.CheckURL = v2CheckURL
	}
	return o
}

func (c *Connect2) Name() string { return "fordconnect" }
func (c *Connect2) Ready() bool  { return c.store.GetKV(refreshKey) != "" }

func asJSON(b []byte) json.RawMessage {
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	s, _ := json.Marshal(string(truncate(b, 2000)))
	return s
}

// Fetch reads telemetry every time, and the slower endpoints (garage, health
// alerts, wallbox, departure times, charge schedules) every slowEvery. Failures
// of the slow ones are recorded, never fatal.
func (c *Connect2) Fetch(ctx context.Context) (*State, error) {
	tel, err := c.get(ctx, "/telemetry")
	if err != nil {
		return nil, err
	}
	o := c.opts()
	want, taken := o.VIN, []string(nil)
	if want == "" && o.Taken != nil {
		taken = o.Taken()
	}
	if seen := c.store.GetKV(SeenVINKey); want == "" && seen != "" && !hasVIN(taken, seen) {
		want = seen // not chosen: keep the vehicle it already follows (Ford's order changes), unless another car has it
	}
	tel, list, chosen := pickVehicle(tel, want, taken) // one vehicle per car, never a mix
	c.noteVehicles(list, chosen)
	doc := map[string]any{"telemetry": asJSON(tel)}

	// At most ONE slow endpoint per reading (the most overdue), so a burst of requests
	// never trips Ford's limit. Each is re-read every slowEvery.
	if e, ok := c.nextSlow(); ok {
		b, status, err := c.call(ctx, e.Path)
		c.mu.Lock()
		switch {
		case err != nil: // includes Ford asking us to wait: try again on a later reading
			c.slowErr[e.Key] = err.Error()
		case status != 200:
			c.slowErr[e.Key] = fmt.Sprintf("HTTP %d", status)
			c.slowAt[e.Key] = time.Now()
		default:
			c.slow[e.Key] = asJSON(b)
			delete(c.slowErr, e.Key)
			c.slowAt[e.Key] = time.Now()
		}
		c.mu.Unlock()
	}
	c.mu.Lock()
	for k, v := range c.slow {
		doc[k] = v
	}
	if len(c.slowErr) > 0 {
		doc["_endpointErrors"] = c.slowErr
	}
	c.mu.Unlock()
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return ParseState(raw, time.Now()), nil
}

// ProbeResult is one endpoint's answer, with location and VIN values removed.
type ProbeResult struct {
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
	Bytes  int    `json:"bytes"`
	Body   any    `json:"body,omitempty"`
	Note   string `json:"note,omitempty"` // plain-words explanation, for example when Ford is limiting requests
}

// Probe calls every endpoint once and returns the redacted answers, so the
// real response shapes can be inspected or shared without exposing the car's
// location. Tokens are never included.
func (c *Connect2) Probe(ctx context.Context) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	paths := []struct{ Key, Path string }{{"telemetry", "/telemetry"}}
	paths = append(paths, slowEndpoints...)
	for _, e := range paths {
		b, status, err := c.call(ctx, e.Path)
		r := ProbeResult{Status: status, Bytes: len(b)}
		var rl *RateLimitError
		if errors.As(err, &rl) {
			r.Note = fmt.Sprintf("Ford is limiting requests. Wait about %d seconds and try again.", int(rl.After.Seconds())+1)
		}
		if err != nil && status != 429 {
			r.Error = err.Error()
		} else {
			r.Body = Redact(b)
		}
		out[e.Key] = r
	}
	return out
}
