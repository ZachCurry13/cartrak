package server

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// linkError carries an HTTP status and a message that is safe to show.
type linkError struct {
	Status int
	Msg    string
}

// pending returns the state value of a link started in the last 15 minutes.
func (s *Server) pending() (string, bool) {
	parts := strings.SplitN(s.S.GetKV("ford_state"), "|", 2)
	if len(parts) != 2 {
		return "", false
	}
	exp, _ := strconv.ParseInt(parts[1], 10, 64)
	if parts[0] == "" || time.Now().Unix() > exp {
		return "", false
	}
	return parts[0], true
}

// finishLink exchanges the one-time code for tokens. A state value, when
// present, must match the one we issued; a pasted bare code has no state, so
// it is accepted only while a link we started is still pending.
func (s *Server) finishLink(ctx context.Context, state, code string) *linkError {
	if s.Link == nil {
		return &linkError{404, "this provider needs no link"}
	}
	want, ok := s.pending()
	if !ok {
		return &linkError{400, "No sign-in is waiting (it may have expired). Press Connect Ford and start again."}
	}
	if state != "" && subtle.ConstantTimeCompare([]byte(want), []byte(state)) != 1 {
		return &linkError{400, "That address belongs to a different sign-in. Press Connect Ford and start again."}
	}
	if code == "" {
		return &linkError{400, "Ford did not return a code."}
	}
	_ = s.S.SetKV("ford_state", "")
	if err := s.Link.Exchange(ctx, code); err != nil {
		return &linkError{502, "Ford refused the code: " + err.Error()}
	}
	s.P.Now()
	return nil
}

// fordCallback handles Ford redirecting the browser straight back to CarTrak
// (when the registered redirect URI is CarTrak's own address).
func (s *Server) fordCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		fail(w, 400, explainFordError(e, q.Get("error_description")))
		return
	}
	if le := s.finishLink(r.Context(), q.Get("state"), q.Get("code")); le != nil {
		fail(w, le.Status, le.Msg)
		return
	}
	http.Redirect(w, r, "/#/settings", http.StatusFound)
}

// fordComplete finishes a link from a pasted address. When the registered
// redirect URI is http://localhost the browser lands on a page that does not
// load, but its address bar holds the code: the admin pastes that address here.
func (s *Server) fordComplete(w http.ResponseWriter, r *http.Request) {
	var b struct {
		URL string `json:"url"`
	}
	if decode(r, &b) != nil || strings.TrimSpace(b.URL) == "" {
		fail(w, 400, "Paste the whole address from the browser's address bar.")
		return
	}
	p := parseReturn(b.URL)
	if p.Err != "" {
		fail(w, 400, explainFordError(p.Err, p.Desc))
		return
	}
	if le := s.finishLink(r.Context(), p.State, p.Code); le != nil {
		fail(w, le.Status, le.Msg)
		return
	}
	ok(w, "linked")
}

type returned struct{ Code, State, Err, Desc string }

// parseReturn accepts the full redirect address, just its query string, or a bare code.
func parseReturn(in string) returned {
	in = strings.TrimSpace(strings.Trim(strings.TrimSpace(in), `"'<>`))
	var q url.Values
	switch {
	case strings.Contains(in, "://"):
		if u, err := url.Parse(in); err == nil {
			q = u.Query()
		}
	case strings.Contains(in, "="):
		q, _ = url.ParseQuery(strings.TrimPrefix(in, "?"))
	default:
		return returned{Code: in}
	}
	return returned{Code: q.Get("code"), State: q.Get("state"), Err: q.Get("error"), Desc: q.Get("error_description")}
}

// explainFordError turns an OAuth error into a plain message, dropping the
// correlation id and timestamp lines Azure adds, and adding a hint for the
// most common mistake: a redirect address that is not registered.
func explainFordError(code, desc string) string {
	if i := strings.IndexAny(desc, "\r\n"); i >= 0 {
		desc = desc[:i]
	}
	msg := "Ford returned an error: " + code
	if desc = strings.TrimSpace(desc); desc != "" {
		msg += " (" + desc + ")"
	}
	if code == "redirect_uri_mismatch" || strings.Contains(desc, "AADB2C90006") || strings.Contains(desc, "not registered") {
		msg += ". The redirect URL set in CarTrak (More > Admin settings) must exactly match a redirect URI registered on your Ford app. If you registered http://localhost, enter exactly that."
	}
	return msg
}
