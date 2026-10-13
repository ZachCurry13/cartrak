// Package tunnel runs the Cloudflare connector (cloudflared) as a child
// process, so remote access can be set up from inside the app: paste a tunnel
// token, name the public address, switch it on.
package tunnel

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type State string

const (
	Off          State = "off"
	NotInstalled State = "not-installed"
	Starting     State = "starting"
	Connected    State = "connected"
	Reconnecting State = "reconnecting"
)

type Status struct {
	State   State  `json:"state"`
	Message string `json:"message"`
	Since   int64  `json:"since"`
}

const maxLog = 200

type Supervisor struct {
	Bin  string   // program to run; default "cloudflared"
	Args []string // default: tunnel --no-autoupdate run (the token goes in TUNNEL_TOKEN)

	mu      sync.Mutex
	cancel  context.CancelFunc
	token   string
	st      Status
	logs    []string
	backoff time.Duration // first restart delay; doubles up to a minute (tests lower it)
}

func New() *Supervisor {
	return &Supervisor{Bin: "cloudflared", Args: []string{"tunnel", "--no-autoupdate", "run"},
		st: Status{State: Off, Since: time.Now().Unix()}, backoff: 5 * time.Second}
}

// Installed reports whether the connector program can be found.
func (s *Supervisor) Installed() bool {
	_, err := exec.LookPath(s.Bin)
	return err == nil
}

func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st
}

// Log returns the most recent connector output, with the token removed.
func (s *Supervisor) Log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.logs...)
}

func (s *Supervisor) set(st State, msg string) {
	s.mu.Lock()
	if s.st.State != st {
		s.st.Since = time.Now().Unix()
	}
	s.st.State, s.st.Message = st, msg
	s.mu.Unlock()
}

// Apply makes the running connector match the settings. It is safe to call
// repeatedly: nothing restarts unless the token changed or it was switched on or off.
func (s *Supervisor) Apply(enabled bool, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !enabled || token == "" {
		s.stopLocked()
		s.st = Status{State: Off, Since: time.Now().Unix()}
		return
	}
	if s.cancel != nil && s.token == token {
		return
	}
	s.stopLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.token = cancel, token
	s.st = Status{State: Starting, Message: "Starting the connector…", Since: time.Now().Unix()}
	s.logs = nil
	go s.run(ctx, token)
}

// Stop ends the connector (used at shutdown).
func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Supervisor) stopLocked() {
	if s.cancel != nil {
		s.cancel()
		s.cancel, s.token = nil, ""
	}
}

func (s *Supervisor) run(ctx context.Context, token string) {
	wait := s.backoff
	for ctx.Err() == nil {
		bin, err := exec.LookPath(s.Bin)
		if err != nil {
			if ctx.Err() != nil { // switched off meanwhile: do not overwrite the Off state
				return
			}
			s.set(NotInstalled, "The Cloudflare connector (cloudflared) is not included in this build. Use the Docker image, or run cloudflared yourself.")
			return
		}
		cmd := exec.CommandContext(ctx, bin, s.Args...)
		// The token goes in the environment, not the command line, so it never shows in a process list.
		cmd.Env = append(os.Environ(), "TUNNEL_TOKEN="+token)
		w := &lineWriter{s: s, token: token}
		cmd.Stdout, cmd.Stderr = w, w
		started := time.Now()
		err = cmd.Run()
		w.flush()
		if ctx.Err() != nil {
			return
		}
		msg := "The connector stopped"
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				msg += " (exit code " + strconv.Itoa(ee.ExitCode()) + ")"
			} else {
				msg += ": " + redact(err.Error(), token)
			}
		}
		s.set(Reconnecting, msg+". Restarting in "+wait.Round(time.Second).String()+"…")
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if time.Since(started) > 2*time.Minute {
			wait = s.backoff // it ran for a while: start the backoff over
		} else if wait *= 2; wait > time.Minute {
			wait = time.Minute
		}
	}
}

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

var tokenLike = regexp.MustCompile(`ey[A-Za-z0-9+/=_-]{20,}`)

// redact removes the tunnel token (and anything shaped like one) from text.
func redact(s, token string) string {
	if token != "" {
		s = strings.ReplaceAll(s, token, "<token>")
	}
	return tokenLike.ReplaceAllString(s, "<token>")
}

var tokenInText = regexp.MustCompile(`ey[A-Za-z0-9+/=_-]{40,}`)

// ExtractToken accepts a bare token, or the command Cloudflare shows
// ("cloudflared service install <token>"), and returns just the token.
func ExtractToken(in string) (string, error) {
	in = strings.TrimSpace(in)
	if m := tokenInText.FindString(in); m != "" {
		return m, nil
	}
	if len(in) >= 40 && !strings.ContainsAny(in, " \t\r\n") {
		return in, nil
	}
	return "", errors.New("that does not look like a Cloudflare tunnel token (it is a long string starting with ey…)")
}

var hostName = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$`)

// NormalizeAddress accepts "https://cartrak.example.com/" or "Cartrak.Example.com" and returns the bare host name.
func NormalizeAddress(in string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(in))
	for _, p := range []string{"https://", "http://"} {
		h = strings.TrimPrefix(h, p)
	}
	if i := strings.IndexAny(h, "/?#"); i >= 0 {
		h = h[:i]
	}
	if !hostName.MatchString(h) {
		return "", errors.New("enter just the public host name, like cartrak.example.com")
	}
	return h, nil
}
