package calendar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Fetch downloads the calendar. The address is a secret (anyone with it can read the calendar), so no error ever contains it.
func Fetch(ctx context.Context, hc *http.Client, address string) (string, error) {
	address = strings.TrimSpace(address)
	if strings.HasPrefix(strings.ToLower(address), "webcal://") {
		address = "https://" + address[len("webcal://"):]
	}
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", fmt.Errorf("the calendar address should start with https:// or webcal://")
	}
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("the calendar address could not be used")
	}
	req.Header.Set("User-Agent", "CarTrak (self-hosted)")
	resp, err := hc.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return "", fmt.Errorf("the calendar did not answer in time")
		}
		return "", fmt.Errorf("the calendar could not be reached")
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404:
		return "", fmt.Errorf("the calendar refused the address (HTTP %d): copy the private address again", resp.StatusCode)
	case resp.StatusCode != 200:
		return "", fmt.Errorf("the calendar answered HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("the calendar could not be read")
	}
	if !strings.Contains(string(b[:min(len(b), 2048)]), "BEGIN:VCALENDAR") {
		return "", fmt.Errorf("that address is not a calendar file: use the private iCal (ICS) address, not the page you view the calendar on")
	}
	return string(b), nil
}
