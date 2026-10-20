package connector

import (
	"context"
	"time"
)

// Check looks for a newer connector and records the answer.
func (m *Manager) Check(ctx context.Context) error {
	rel, err := m.latest(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.info.CheckedAt = time.Now().Unix()
	if err != nil {
		m.info.Error = err.Error()
		return err
	}
	m.info.Error, m.info.Latest = "", rel.Version
	m.info.Failed = m.KV.GetKV(kvFailed)
	m.info.Available = m.info.Current != "" && Newer(rel.Version, m.info.Current) && rel.Version != m.info.Failed
	return nil
}

// Run restores the connector chosen last time, then checks once a day until ctx ends.
func (m *Manager) Run(ctx context.Context) {
	m.Restore()
	for {
		if m.Enabled == nil || m.Enabled() {
			c, cancel := context.WithTimeout(ctx, time.Minute)
			_ = m.Check(c)
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(24 * time.Hour):
		}
	}
}
