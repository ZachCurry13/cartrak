package connector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const maxDownload = 150 << 20

var errBusy = errors.New("an update is already running")

// downloadHosts are the only places a connector may be fetched from: GitHub and its release file servers.
var downloadHosts = map[string]bool{
	"github.com": true, "objects.githubusercontent.com": true,
	"release-assets.githubusercontent.com": true, "github-releases.githubusercontent.com": true,
}

func (m *Manager) checkURL(u *url.URL) error {
	if u.Scheme != "https" && !m.AllowInsecure {
		return fmt.Errorf("refusing a download that is not https")
	}
	if m.AllowHost != nil {
		if !m.AllowHost(u.Hostname()) {
			return fmt.Errorf("refusing a download from %s", u.Hostname())
		}
		return nil
	}
	if !downloadHosts[u.Hostname()] {
		return fmt.Errorf("refusing a download from %s", u.Hostname())
	}
	return nil
}

// Restore uses the connector installed from the app last time, if it is still there, runs, and is not older
// than the one that came with this version of CarTrak, and records which version is in use. Call it once at
// start-up, before the tunnel starts.
func (m *Manager) Restore() {
	bundled := m.Sup.Program()
	bundledV, _ := RunVersion(bundled)
	use, useV := bundled, bundledV
	if p := m.KV.GetKV(kvBin); p != "" {
		v, err := RunVersion(p)
		if _, serr := os.Stat(p); serr != nil || err != nil || (bundledV != "" && Newer(bundledV, v)) {
			_ = m.KV.SetKV(kvBin, "") // gone, broken, or the image now ships a newer one
		} else {
			use, useV = p, v
		}
	}
	m.Sup.SetProgram(use)
	m.mu.Lock()
	m.info.Current = useV
	m.mu.Unlock()
}

func (m *Manager) fail(msg string) error {
	m.progress("failed", msg)
	return errors.New(msg)
}

// download fetches the release file into dest and checks its SHA-256 against the published digest.
func (m *Manager) download(ctx context.Context, rel *release, dest string) error {
	u, err := url.Parse(rel.URL)
	if err != nil {
		return err
	}
	if err := m.checkURL(u); err != nil {
		return err
	}
	c := *m.client()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return fmt.Errorf("too many redirects")
		}
		return m.checkURL(req.URL)
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", rel.URL, nil)
	req.Header.Set("User-Agent", "CarTrak connector update")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("the download answered HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxDownload+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > maxDownload {
		return fmt.Errorf("the download was larger than expected")
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != rel.Digest {
		return fmt.Errorf("the download did not match the checksum GitHub published")
	}
	return nil
}
