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
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// Update installs the newest connector and switches to it. If the tunnel does not connect within the grace
// period the previous connector is put back and that version is not offered again. Progress is in Info().
func (m *Manager) Update(ctx context.Context) error {
	m.mu.Lock()
	if m.busy {
		m.mu.Unlock()
		return errBusy
	}
	m.busy = true
	m.mu.Unlock()
	defer func() { m.mu.Lock(); m.busy = false; m.mu.Unlock() }()

	if !m.Info().Supported {
		return m.fail("Installing a connector from the app works only on the Linux Docker image.")
	}
	m.progress("working", "Looking up the newest version…")
	rel, err := m.latest(ctx)
	if err != nil {
		return m.fail("Could not look up the newest version: " + err.Error())
	}
	cur := m.Info().Current
	if cur != "" && !Newer(rel.Version, cur) {
		return m.fail("The connector is already up to date.")
	}
	if len(rel.Digest) != 64 {
		return m.fail("GitHub did not publish a checksum for this file, so CarTrak will not install it.")
	}
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return m.fail("Could not create the folder for the connector: " + err.Error())
	}
	part := filepath.Join(m.Dir, "download.part"+m.ExeSuffix)
	defer os.Remove(part)
	m.progress("working", "Downloading version "+rel.Version+"…")
	if err := m.download(ctx, rel, part); err != nil {
		return m.fail("Download failed: " + err.Error())
	}
	m.progress("working", "Checking it runs…")
	if v, err := RunVersion(part); err != nil {
		return m.fail("The new connector " + err.Error() + ". If the data folder does not allow running programs, it cannot be installed here.")
	} else if v != rel.Version {
		return m.fail("The new connector reports version " + v + ", not " + rel.Version + ".")
	}
	final := filepath.Join(m.Dir, "cloudflared-"+rel.Version+m.ExeSuffix)
	if err := os.Rename(part, final); err != nil {
		return m.fail("Could not save the connector: " + err.Error())
	}

	prev, wasRunning := m.Sup.Program(), m.Sup.Running()
	m.Sup.SetProgram(final)
	if wasRunning {
		m.progress("working", "Switching over. Remote access drops for a few seconds…")
		m.Sup.Restart()
		grace := m.Grace
		if grace == 0 {
			grace = 90 * time.Second
		}
		if !m.Sup.WaitConnected(grace) {
			m.Sup.SetProgram(prev)
			m.Sup.Restart()
			_ = m.KV.SetKV(kvFailed, rel.Version)
			_ = os.Remove(final)
			m.mu.Lock()
			m.info.Failed, m.info.Available = rel.Version, false
			m.mu.Unlock()
			return m.fail("Version " + rel.Version + " did not connect to Cloudflare within " + grace.Round(time.Second).String() + ", so the previous connector was put back.")
		}
	}
	_ = m.KV.SetKV(kvPrev, prev)
	_ = m.KV.SetKV(kvBin, final)
	_ = m.KV.SetKV(kvFailed, "")
	m.prune(final, prev)
	m.mu.Lock()
	m.info.Current, m.info.Available, m.info.Failed = rel.Version, false, ""
	m.mu.Unlock()
	if wasRunning {
		m.progress("done", "Updated to "+rel.Version+". The tunnel is connected.")
	} else {
		m.progress("done", "Installed "+rel.Version+". It will be used when remote access is switched on.")
	}
	return nil
}

// prune keeps the two newest installed connectors plus the ones in use.
func (m *Manager) prune(keep ...string) {
	files, _ := filepath.Glob(filepath.Join(m.Dir, "cloudflared-*"))
	sort.Slice(files, func(i, j int) bool {
		return Newer(m.versionOf(files[i]), m.versionOf(files[j]))
	})
	protected := map[string]bool{}
	for _, k := range keep {
		protected[k] = true
	}
	for i, f := range files {
		if i >= 2 && !protected[f] {
			_ = os.Remove(f)
		}
	}
}

func (m *Manager) versionOf(file string) string {
	return strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "cloudflared-"), m.ExeSuffix)
}
