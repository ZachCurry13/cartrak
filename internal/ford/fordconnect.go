package ford

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIBase  = "https://api.mps.ford.com"
	defaultTokenURL = "https://dah2vb2cprod.b2clogin.com/914d88b1-3523-4bf6-9be4-1b96b4f6f919/oauth2/v2.0/token?p=B2C_1A_signup_signin_common"
	linkBase        = "https://fordconnect.cv.ford.com/"
	apiVersion      = "2020-06-01"
)

// ErrNotLinked means the owner has not completed the Ford account link yet.
var ErrNotLinked = errors.New("ford account not linked")

// TokenStore keeps the OAuth refresh token between restarts.
type TokenStore interface {
	GetKV(k string) string
	SetKV(k, v string) error
}

type ConnectOpts struct {
	ClientID, ClientSecret, ApplicationID, RedirectURL, VehicleID string
	APIBase, TokenURL                                             string // override for tests
}

type Connect struct {
	optsFn func() ConnectOpts // read on every call so admin-panel changes apply without a restart
	vid    string             // vehicle picked automatically from the authorized list
	store  TokenStore
	hc     *http.Client
	mu     sync.Mutex
	acc    string
	exp    time.Time
}

// NewConnect uses fixed options.
func NewConnect(o ConnectOpts, st TokenStore) *Connect {
	return NewConnectFn(func() ConnectOpts { return o }, st)
}

// NewConnectFn re-reads its options (credentials, redirect URL...) on each use.
func NewConnectFn(fn func() ConnectOpts, st TokenStore) *Connect {
	return &Connect{optsFn: fn, store: st, hc: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Connect) opts() ConnectOpts {
	o := c.optsFn()
	if o.APIBase == "" {
		o.APIBase = defaultAPIBase
	}
	if o.TokenURL == "" {
		o.TokenURL = defaultTokenURL
	}
	return o
}

func (c *Connect) Name() string { return "fordconnect" }
func (c *Connect) Ready() bool  { return c.store.GetKV("ford_refresh") != "" }

// LinkURL is where the owner signs in to FordPass and authorizes the vehicle.
func (c *Connect) LinkURL(state string) string {
	q := url.Values{
		"make": {"F"}, "application_id": {c.opts().ApplicationID}, "client_id": {c.opts().ClientID},
		"response_type": {"code"}, "state": {state}, "redirect_uri": {c.opts().RedirectURL},
		"scope": {"access"},
	}
	return linkBase + "?" + q.Encode()
}

// Exchange trades the one-time code from the link redirect for tokens.
func (c *Connect) Exchange(ctx context.Context, code string) error {
	c.mu.Lock()
	c.vid = "" // a new link may be a different account or vehicle
	c.mu.Unlock()
	return c.token(ctx, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {c.opts().RedirectURL},
	})
}

func (c *Connect) token(ctx context.Context, form url.Values) error {
	form.Set("client_id", c.opts().ClientID)
	form.Set("client_secret", c.opts().ClientSecret)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.opts().TokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return fmt.Errorf("ford token: HTTP %d", resp.StatusCode) // body omitted: may echo secrets
	}
	var t struct {
		Access    string      `json:"access_token"`
		Refresh   string      `json:"refresh_token"`
		ExpiresIn json.Number `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &t); err != nil || t.Access == "" {
		return errors.New("ford token: bad response")
	}
	secs, _ := t.ExpiresIn.Int64()
	if secs <= 0 {
		secs = 1200
	}
	c.acc, c.exp = t.Access, time.Now().Add(time.Duration(secs)*time.Second)
	if t.Refresh != "" {
		return c.store.SetKV("ford_refresh", t.Refresh)
	}
	return nil
}

func (c *Connect) bearer(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.acc != "" && time.Until(c.exp) > time.Minute {
		return c.acc, nil
	}
	rt := c.store.GetKV("ford_refresh")
	if rt == "" {
		return "", ErrNotLinked
	}
	if err := c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt}}); err != nil {
		return "", err
	}
	return c.acc, nil
}

func (c *Connect) get(ctx context.Context, path string) (json.RawMessage, error) {
	tok, err := c.bearer(ctx)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.opts().APIBase+"/api/fordconnect/vehicles/v1"+path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Application-Id", c.opts().ApplicationID)
	req.Header.Set("api-version", apiVersion)
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == 401 {
		c.mu.Lock()
		c.acc = "" // force a refresh on the next call
		c.mu.Unlock()
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ford %s: HTTP %d", path, resp.StatusCode)
	}
	return b, nil
}

// vehicleID returns the configured id, else the first authorized vehicle.
func (c *Connect) vehicleID(ctx context.Context) (string, error) {
	if v := c.opts().VehicleID; v != "" {
		return v, nil
	}
	if c.vid != "" {
		return c.vid, nil
	}
	b, err := c.get(ctx, "")
	if err != nil {
		return "", err
	}
	var l struct {
		Vehicles []struct {
			ID   string `json:"vehicleId"`
			Auth int    `json:"vehicleAuthorizationIndicator"`
		} `json:"vehicles"`
	}
	if json.Unmarshal(b, &l) == nil {
		for _, v := range l.Vehicles {
			if v.Auth == 1 {
				c.vid = v.ID
				return v.ID, nil
			}
		}
	}
	return "", errors.New("no authorized vehicle on the linked Ford account")
}

func (c *Connect) Fetch(ctx context.Context) (*State, error) {
	id, err := c.vehicleID(ctx)
	if err != nil {
		return nil, err
	}
	info, err := c.get(ctx, "/"+id)
	if err != nil {
		return nil, err
	}
	merged := map[string]json.RawMessage{"info": info}
	if loc, err := c.get(ctx, "/"+id+"/location"); err == nil {
		merged["location"] = loc
	}
	raw, _ := json.Marshal(merged)
	return ParseState(raw, time.Now()), nil
}
