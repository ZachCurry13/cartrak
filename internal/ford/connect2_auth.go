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
	"time"
)

func (c *Connect2) scope(o Connect2Opts) string { return o.ClientID + " offline_access openid" }

// LinkURL is where the owner signs in to FordPass and authorizes the vehicle.
func (c *Connect2) LinkURL(state string) string {
	o := c.opts()
	return c.linkURL(o, o.RedirectURL, state)
}

// Exchange trades the one-time code from the sign-in redirect for tokens.
func (c *Connect2) Exchange(ctx context.Context, code string) error {
	o := c.opts()
	return c.token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {o.RedirectURL}})
}

func (c *Connect2) token(ctx context.Context, form url.Values) error {
	o := c.opts()
	form.Set("client_id", o.ClientID)
	form.Set("client_secret", o.ClientSecret)
	// Ford's Postman collection sends a scope; other working clients send none. Try the
	// documented form first, and if Ford answers 400 try again without it.
	status, b, err := c.postToken(ctx, o, form, true)
	if err == nil && status == http.StatusBadRequest {
		status, b, err = c.postToken(ctx, o, form, false)
	}
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("ford token: HTTP %d%s", status, oauthReason(b)) // never echo the body: it may hold secrets
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
	c.mu.Lock()
	c.acc, c.exp = t.Access, time.Now().Add(time.Duration(secs)*time.Second)
	c.mu.Unlock()
	if t.Refresh != "" {
		return c.store.SetKV(refreshKey, t.Refresh)
	}
	return nil
}

// oauthReason returns the short OAuth error code (for example invalid_grant),
// which is safe to show; the rest of an error body is dropped.
func oauthReason(b []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(b, &e) == nil && e.Error != "" && len(e.Error) < 60 && !strings.ContainsAny(e.Error, " \n") {
		return " (" + e.Error + ")"
	}
	return ""
}

func (c *Connect2) bearer(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.acc != "" && time.Until(c.exp) > time.Minute {
		defer c.mu.Unlock()
		return c.acc, nil
	}
	c.mu.Unlock()
	rt := c.store.GetKV(refreshKey)
	if rt == "" {
		return "", ErrNotLinked
	}
	if err := c.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt}, "redirect_uri": {c.opts().RedirectURL}}); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.acc, nil
}

// postToken sends one token request and returns the status and body.
func (c *Connect2) postToken(ctx context.Context, o Connect2Opts, form url.Values, withScope bool) (int, []byte, error) {
	f := url.Values{}
	for k, v := range form {
		f[k] = v
	}
	if withScope {
		f.Set("scope", c.scope(o))
	}
	req, err := http.NewRequestWithContext(ctx, "POST", o.TokenURL, strings.NewReader(f.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, b, nil
}
