package tunnel

import (
	"bytes"
	"strings"
)

// lineWriter turns connector output into log lines and watches for connection state.
type lineWriter struct {
	s     *Supervisor
	token string
	buf   []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, 10) // newline
		if i < 0 {
			break
		}
		w.line(string(w.buf[:i]))
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.line(string(w.buf))
		w.buf = nil
	}
}

func (w *lineWriter) line(l string) {
	l = strings.TrimRight(redact(l, w.token), "\r")
	if strings.TrimSpace(l) == "" {
		return
	}
	w.s.mu.Lock()
	w.s.logs = append(w.s.logs, l)
	if len(w.s.logs) > maxLog {
		w.s.logs = w.s.logs[len(w.s.logs)-maxLog:]
	}
	w.s.mu.Unlock()
	switch {
	case strings.Contains(l, "Registered tunnel connection"):
		w.s.set(Connected, "Connected to Cloudflare.")
	case strings.Contains(l, "Unregistered tunnel connection") || strings.Contains(l, "Retrying connection"):
		if w.s.Status().State == Connected {
			w.s.set(Reconnecting, "Lost the connection to Cloudflare; reconnecting…")
		}
	case strings.Contains(l, "Unauthorized") || strings.Contains(l, "invalid token") || strings.Contains(l, "Invalid tunnel secret"):
		w.s.set(Reconnecting, "Cloudflare rejected the token. Paste a fresh token from the tunnel's page.")
	}
}
