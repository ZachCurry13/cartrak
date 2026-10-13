package ford

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIBase  = "https://api.mps.ford.com"
	defaultTokenURL = "https://dah2vb2cprod.b2clogin.com/914d88b1-3523-4bf6-9be4-1b96b4f6f919/oauth2/v2.0/token?p=B2C_1A_signup_signin_common"
	linkBase        = "https://fordconnect.cv.ford.com/common/login/"
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
	if resp.StatusCode == 429 {
		return nil, &RateLimitError{After: retryAfter(resp.Header.Get("Retry-After"))}
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ford %s: HTTP %d", path, resp.StatusCode)
	}
	return b, nil
}

// retryAfter reads a Retry-After header given in seconds (the usual form).
func retryAfter(v string) time.Duration {
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 && n < 3600 {
		return time.Duration(n) * time.Second
	}
	return 0
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
