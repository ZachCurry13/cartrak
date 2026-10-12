// Package updater asks GitHub (public releases list only) whether a newer
// CarTrak version exists. It sends no vehicle or personal data.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultURL = "https://api.github.com/repos/ZachCurry13/cartrak/releases/latest"

type Info struct {
	Latest    string `json:"latest"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
	CheckedAt int64  `json:"checkedAt"`
	Error     string `json:"error"`
}

type Checker struct {
	Current string
	URL     string // defaults to DefaultURL; overridden in tests
	HC      *http.Client

	mu   sync.Mutex
	info Info
}

func New(current string) *Checker {
	return &Checker{Current: current, URL: DefaultURL, HC: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Checker) Info() Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.info
}

// Check fetches the latest release and records the result.
func (c *Checker) Check(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cartrak/"+c.Current)
	resp, err := c.HC.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("GitHub answered HTTP %d", resp.StatusCode)
		}
	}
	var rel struct {
		Tag string `json:"tag_name"`
		URL string `json:"html_url"`
	}
	if err == nil {
		err = json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&rel)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.info.CheckedAt = time.Now().Unix()
	if err != nil {
		c.info.Error = err.Error()
		return err
	}
	c.info.Error = ""
	c.info.Latest = strings.TrimPrefix(rel.Tag, "v")
	c.info.URL = rel.URL
	c.info.Available = Newer(c.info.Latest, c.Current)
	return nil
}

// Run checks at start and then daily while enabled() is true, until ctx ends.
func (c *Checker) Run(ctx context.Context, enabled func() bool) {
	t := time.NewTimer(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if enabled() {
				_ = c.Check(ctx)
			}
			t.Reset(24 * time.Hour)
		}
	}
}

// Newer reports whether latest is a higher x.y.z than current. A current
// version that is not x.y.z (such as "dev") never counts as outdated.
func Newer(latest, current string) bool {
	l, ok1 := parse(latest)
	c, ok2 := parse(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3)
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		if j := strings.IndexAny(p, "-+"); j >= 0 {
			p = p[:j]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
