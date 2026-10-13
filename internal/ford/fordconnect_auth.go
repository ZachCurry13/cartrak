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
