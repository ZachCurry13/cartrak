// Package bugreport sends a bug report from the app to a private GitHub repository: the owner's note, what the page looked like
// (a screenshot, its text and the screen it was on), and the debug report. Each report becomes a folder of files and an issue that
// points at it, so it can be read later with the gh tool. The token only ever goes in the Authorization header.
package bugreport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Report is everything that is sent.
type Report struct {
	Version    string // the app version
	Note       string // what the person typed
	Page       string // the page, such as "#/trips"
	Device     string // the screen and browser it was on, in words
	PageText   string // the visible text of the page
	Debug      string // the debug report, or ""
	Screenshot []byte // a PNG of the page, or nil
}

// Result says where it went.
type Result struct {
	ID     string `json:"id"`
	Number int    `json:"number"` // the issue number
	URL    string `json:"url"`
}

// Client writes to one repository.
type Client struct {
	Repo  string // owner/name
	Token string
	API   string // https://api.github.com unless a test says otherwise
	HTTP  *http.Client
}

func (c *Client) api() string {
	if c.API != "" {
		return strings.TrimRight(c.API, "/")
	}
	return "https://api.github.com"
}

func (c *Client) do(ctx context.Context, method, path string, body any) (map[string]any, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.api()+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "cartrak")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach GitHub")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if resp.StatusCode/100 != 2 {
		msg, _ := out["message"].(string)
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("GitHub did not accept the token (it may have expired)")
		case 403, 404:
			return nil, fmt.Errorf("GitHub said %d: check that the token can write Contents and Issues in %s (%s)", resp.StatusCode, c.Repo, msg)
		}
		return nil, fmt.Errorf("GitHub said %d: %s", resp.StatusCode, msg)
	}
	return out, nil
}

func newID(now time.Time) string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

// put creates one file in the report's folder.
func (c *Client) put(ctx context.Context, id, name string, data []byte) (string, error) {
	path := fmt.Sprintf("/repos/%s/contents/reports/%s/%s", c.Repo, url.PathEscape(id), url.PathEscape(name))
	out, err := c.do(ctx, http.MethodPut, path, map[string]string{"message": "report " + id + ": " + name, "content": base64.StdEncoding.EncodeToString(data)})
	if err != nil {
		return "", err
	}
	if cont, ok := out["content"].(map[string]any); ok {
		if u, _ := cont["html_url"].(string); u != "" {
			return u, nil
		}
	}
	return "", nil
}

func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// Send writes the files and opens the issue.
func (c *Client) Send(ctx context.Context, r Report, now time.Time) (Result, error) {
	id := newID(now)
	note := fmt.Sprintf("# Bug report %s\n\n%s\n\n- App: v%s\n- Page: %s\n- Screen: %s\n\n## Page text\n\n```\n%s\n```\n", id, strings.TrimSpace(r.Note), r.Version, r.Page, r.Device, strings.ReplaceAll(r.PageText, "```", "'''"))
	noteURL, err := c.put(ctx, id, "report.md", []byte(note))
	if err != nil {
		return Result{ID: id}, err
	}
	var files []string
	if r.Debug != "" {
		if _, err := c.put(ctx, id, "debug.txt", []byte(r.Debug)); err != nil {
			return Result{ID: id}, err
		}
		files = append(files, "debug.txt")
	}
	if len(r.Screenshot) > 0 {
		if _, err := c.put(ctx, id, "screen.png", r.Screenshot); err != nil {
			return Result{ID: id}, err
		}
		files = append(files, "screen.png")
	}
	title := firstLine(r.Note, 70)
	if title == "" {
		title = "Bug report"
	}
	body := fmt.Sprintf("%s\n\n---\nApp v%s · %s · %s\n\nFiles: `reports/%s/` (report.md%s)\n%s", strings.TrimSpace(r.Note), r.Version, r.Page, r.Device, id, joinFiles(files), noteURL)
	out, err := c.do(ctx, http.MethodPost, "/repos/"+c.Repo+"/issues", map[string]any{"title": fmt.Sprintf("[v%s] %s", r.Version, title), "body": body})
	if err != nil {
		return Result{ID: id}, err
	}
	res := Result{ID: id}
	if n, ok := out["number"].(float64); ok {
		res.Number = int(n)
	}
	res.URL, _ = out["html_url"].(string)
	return res, nil
}

func joinFiles(f []string) string {
	if len(f) == 0 {
		return ""
	}
	return ", " + strings.Join(f, ", ")
}

// Check says whether the repository can take reports: it exists, the token can write to it, and it is private. A public
// repository is refused because a report holds screenshots and positions.
func (c *Client) Check(ctx context.Context) error {
	out, err := c.do(ctx, "GET", "/repos/"+c.Repo, nil)
	if err != nil {
		return err
	}
	if private, _ := out["private"].(bool); !private {
		return fmt.Errorf("%s is public: reports contain screenshots and positions, so make it private (GitHub, the repository's Settings, Danger Zone, Change visibility)", c.Repo)
	}
	if perms, ok := out["permissions"].(map[string]any); ok {
		if push, _ := perms["push"].(bool); !push {
			return fmt.Errorf("the token cannot write to %s: set Contents and Issues to Read and write", c.Repo)
		}
	}
	return nil
}
