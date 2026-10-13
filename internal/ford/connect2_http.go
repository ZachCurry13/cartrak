package ford

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// callRaw performs one GET and returns the body and status without judging it.
// Use call, which also respects Ford's request limits.
func (c *Connect2) callRaw(ctx context.Context, path string) ([]byte, int, error) {
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
		d := errorDetail(b)
		if status >= 500 {
			return nil, &ServerError{Path: path, Status: status, Detail: d}
		}
		if d != "" {
			return nil, fmt.Errorf("ford %s: HTTP %d (Ford said: %s)", path, status, d)
		}
		return nil, fmt.Errorf("ford %s: HTTP %d", path, status)
	}
	return b, nil
}
