// Package connector keeps the Cloudflare connector (cloudflared) up to date without breaking remote access.
// The Docker image ships one version. When Cloudflare publishes a newer one, an admin can install it from
// the app: it is downloaded from GitHub over https, its SHA-256 is checked against the digest GitHub
// publishes, it must run, and the tunnel must come back within the grace period or the old one is restored.
package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZachCurry13/cartrak/internal/tunnel"
)

const (
	DefaultAPI = "https://api.github.com/repos/cloudflare/cloudflared/releases/latest"

	kvBin    = "connector_bin"    // path of the connector installed from the app
	kvPrev   = "connector_prev"   // the one that was in use before it
	kvFailed = "connector_failed" // a version that did not connect, so it is not offered again
)

// KV is the small key-value store the choices are kept in.
type KV interface {
	GetKV(k string) string
	SetKV(k, v string) error
}

// Info is what the app shows.
type Info struct {
	Supported bool     `json:"supported"` // a Linux build that can install a connector
	Current   string   `json:"current"`
	Latest    string   `json:"latest"`
	Available bool     `json:"available"`
	CheckedAt int64    `json:"checkedAt"`
	Error     string   `json:"error"`
	Failed    string   `json:"failed"` // a version that was tried and rolled back
	Update    Progress `json:"update"`
}

// Progress is how an update is going.
type Progress struct {
	State   string `json:"state"` // idle, working, done, failed
	Message string `json:"message"`
	Since   int64  `json:"since"`
}

type release struct {
	Version string
	URL     string
	Digest  string // hex SHA-256
}

type Manager struct {
	Sup     *tunnel.Supervisor
	Dir     string // where installed connectors are kept
	KV      KV
	Enabled func() bool // daily checks; nil means on
	API     string
	HC      *http.Client
	Grace   time.Duration // how long a new connector has to connect; default 90 s

	// Tests change these.
	GOOS, GOARCH  string
	AllowHost     func(host string) bool
	AllowInsecure bool
	ExeSuffix     string // ".exe" when the tests run on Windows

	mu   sync.Mutex
	info Info
	busy bool
}

func (m *Manager) goos() string {
	if m.GOOS != "" {
		return m.GOOS
	}
	return runtime.GOOS
}

func (m *Manager) goarch() string {
	if m.GOARCH != "" {
		return m.GOARCH
	}
	return runtime.GOARCH
}

func (m *Manager) api() string {
	if m.API != "" {
		return m.API
	}
	return DefaultAPI
}

func (m *Manager) client() *http.Client {
	if m.HC != nil {
		return m.HC
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

// Info returns the latest known state.
func (m *Manager) Info() Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.info
	i.Supported = m.goos() == "linux" && (m.goarch() == "amd64" || m.goarch() == "arm64")
	if i.Update.State == "" {
		i.Update.State = "idle"
	}
	return i
}

func (m *Manager) progress(state, msg string) {
	m.mu.Lock()
	m.info.Update = Progress{State: state, Message: msg, Since: time.Now().Unix()}
	m.mu.Unlock()
}

var versionRe = regexp.MustCompile(`version\s+(\d+(?:\.\d+)+)`)

// RunVersion asks a connector program for its version, for example "2026.10.0".
func RunVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("it would not run: %v", err)
	}
	if m := versionRe.FindSubmatch(out); m != nil {
		return string(m[1]), nil
	}
	return "", fmt.Errorf("it did not say its version")
}

// Newer reports whether version a is higher than b ("2026.10.1" against "2026.10.0").
func Newer(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// latest reads Cloudflare's newest release and picks the Linux file for this machine.
func (m *Manager) latest(ctx context.Context) (*release, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", m.api(), nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CarTrak connector check")
	resp, err := m.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub answered HTTP %d", resp.StatusCode)
	}
	var rel struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	want := "cloudflared-linux-" + m.goarch()
	for _, a := range rel.Assets {
		if a.Name == want {
			return &release{Version: strings.TrimPrefix(rel.Tag, "v"), URL: a.URL, Digest: strings.ToLower(strings.TrimPrefix(a.Digest, "sha256:"))}, nil
		}
	}
	return nil, fmt.Errorf("release %s has no %s file", rel.Tag, want)
}
