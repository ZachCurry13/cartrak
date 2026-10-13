package ford

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// FordConnect 2.0: bearer-token queries under /fcon-query/v1 and an Azure B2C
// sign-in. Addresses come from Ford's FordConnect 2.0 Postman collection. The
// sign-in (authorize) address is not in that collection: it follows the same
// pattern as the token address, and can be overridden (FORD_AUTH_URL).
const (
	v2Host      = "https://api.vehicle.ford.com"
	v2TokenURL  = v2Host + "/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token?p=B2C_1A_FCON_AUTHORIZE"
	v2AuthURL   = v2Host + "/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/authorize?p=B2C_1A_FCON_AUTHORIZE"
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
	APIBase, TokenURL                            string // overridden in tests
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
	slowAt  time.Time
}

func NewConnect2(fn func() Connect2Opts, st TokenStore) *Connect2 {
	return &Connect2{optsFn: fn, store: st, hc: &http.Client{Timeout: 30 * time.Second},
		slow: map[string]json.RawMessage{}, slowErr: map[string]string{}}
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
		o.AuthURL = v2AuthURL
	}
	return o
}

func (c *Connect2) Name() string { return "fordconnect" }
func (c *Connect2) Ready() bool  { return c.store.GetKV(refreshKey) != "" }

// call performs one GET and returns the body and status without judging it.
func (c *Connect2) call(ctx context.Context, path string) ([]byte, int, error) {
	tok, err := c.bearer(ctx)
	if err != nil {
		return nil, 0, err
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.opts().APIBase+path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode == 401 {
		c.mu.Lock()
		c.acc = "" // force a refresh next time
		c.mu.Unlock()
	}
	if resp.StatusCode == 429 {
		return b, 429, &RateLimitError{After: retryAfter(resp.Header.Get("Retry-After"))}
	}
	return b, resp.StatusCode, nil
}

func (c *Connect2) get(ctx context.Context, path string) ([]byte, error) {
	b, status, err := c.call(ctx, path)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ford %s: HTTP %d", path, status)
	}
	return b, nil
}

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
	doc := map[string]any{"telemetry": asJSON(tel)}

	c.mu.Lock()
	due := time.Since(c.slowAt) > slowEvery
	c.mu.Unlock()
	if due {
		got, errs := map[string]json.RawMessage{}, map[string]string{}
		for _, e := range slowEndpoints {
			b, status, err := c.call(ctx, e.Path)
			switch {
			case err != nil:
				errs[e.Key] = err.Error()
			case status != 200:
				errs[e.Key] = fmt.Sprintf("HTTP %d", status)
			default:
				got[e.Key] = asJSON(b)
			}
		}
		c.mu.Lock()
		c.slow, c.slowErr, c.slowAt = got, errs, time.Now()
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
		if err != nil && status != 429 {
			r.Error = err.Error()
		} else {
			r.Body = Redact(b)
		}
		out[e.Key] = r
	}
	return out
}
