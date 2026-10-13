package ford

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// authURL is Ford's sign-in address for a given redirect address.
func (c *Connect2) authURL(o Connect2Opts, redirect, state string) string {
	if strings.Contains(o.AuthURL, "{") { // a link copied from Ford documentation, with placeholders
		return strings.NewReplacer("{client_id}", url.QueryEscape(o.ClientID), "{redirect_uri}", url.QueryEscape(redirect),
			"{state}", url.QueryEscape(state), "{scope}", url.QueryEscape(c.scope(o))).Replace(o.AuthURL)
	}
	q := url.Values{"client_id": {o.ClientID}, "response_type": {"code"}, "redirect_uri": {redirect},
		"scope": {c.scope(o)}, "state": {state}, "response_mode": {"query"}}
	sep := "?"
	if strings.Contains(o.AuthURL, "?") {
		sep = "&"
	}
	return o.AuthURL + sep + q.Encode()
}

// RedirectCheck is Ford's answer for one candidate redirect address.
type RedirectCheck struct {
	URI      string `json:"uri"`
	Accepted bool   `json:"accepted"`
	Detail   string `json:"detail,omitempty"`
}

// CheckRedirect asks Ford's sign-in service whether it accepts redirectURI for
// this client. Nothing is signed in and no secret is sent: Ford answers a
// registered address with its sign-in flow, and an unregistered one with a
// redirect_uri_mismatch (AADB2C90006) error.
func (c *Connect2) CheckRedirect(ctx context.Context, redirectURI string) (RedirectCheck, error) {
	o := c.opts()
	res := RedirectCheck{URI: redirectURI}
	req, err := http.NewRequestWithContext(ctx, "GET", c.authURL(o, redirectURI, "cartrak-check"), nil)
	if err != nil {
		return res, err
	}
	hc := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := hc.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	desc := ""
	if u, err := url.Parse(loc); err == nil {
		desc = u.Query().Get("error_description")
		if e := u.Query().Get("error"); e == "redirect_uri_mismatch" || strings.Contains(desc, "AADB2C90006") {
			res.Detail = "not registered on your Ford app"
			return res, nil
		}
	}
	if resp.StatusCode >= 500 {
		return res, fmt.Errorf("Ford answered HTTP %d", resp.StatusCode)
	}
	res.Accepted = true
	if desc != "" { // accepted, but Ford reported a different problem for a request with no signed-in user
		if i := strings.IndexAny(desc, "\r\n"); i >= 0 {
			desc = desc[:i]
		}
		res.Detail = "Ford accepts this address (it reported: " + desc + ")"
	}
	return res, nil
}
