package connector

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
