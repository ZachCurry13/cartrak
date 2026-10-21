package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZachCurry13/cartrak/internal/store"
)

// describeAgent turns a browser's long User-Agent text into something readable, like "Chrome on Android".
func describeAgent(ua string) string {
	os := "unknown device"
	for _, c := range []struct{ key, name string }{{"Android", "Android"}, {"iPhone", "iPhone"}, {"iPad", "iPad"}, {"Windows", "Windows"}, {"Mac OS X", "Mac"}, {"Linux", "Linux"}} {
		if strings.Contains(ua, c.key) {
			os = c.name
			break
		}
	}
	browser := "browser"
	for _, c := range []struct{ key, name string }{{"Edg/", "Edge"}, {"OPR/", "Opera"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"CriOS", "Chrome"}, {"Safari/", "Safari"}} {
		if strings.Contains(ua, c.key) {
			browser = c.name
			break
		}
	}
	if ua == "" {
		return "unknown"
	}
	return browser + " on " + os
}

// logSignin writes a line to the sign-in log. The name is only what was typed, cut short.
func (s *Server) logSignin(r *http.Request, name string, ok bool) {
	if len(name) > 60 {
		name = name[:60]
	}
	_ = s.core().RecordSignin(name, ok, s.clientIP(r), describeAgent(r.UserAgent()))
}

// securityInfo is the sign-in log and the devices signed in right now. Admin only.
func (s *Server) securityInfo(w http.ResponseWriter, r *http.Request) {
	logs, err := s.core().RecentSignins(100)
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	sessions, err := s.core().ListSessions()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	cur := ""
	if c, err := r.Cookie(cookieName); err == nil {
		cur = c.Value
	}
	out := make([]sessionRow, 0, len(sessions))
	for _, x := range sessions {
		out = append(out, sessionRow{x.ID, x.User, x.Role, x.Created, x.Expires, x.IP, x.Agent, x.Token == cur, max(x.LastActive, x.Created)})
	}
	users, _ := s.core().ListUsers()
	ok(w, map[string]any{"signins": logs, "sessions": out, "people": peopleSignedIn(users, out, logs), "now": time.Now().Unix()})
}

type sessionRow struct {
	ID         int64  `json:"id"`
	User       string `json:"user"`
	Role       string `json:"role"`
	Created    int64  `json:"created"`
	Expires    int64  `json:"expires"`
	IP         string `json:"ip"`
	Agent      string `json:"agent"`
	Current    bool   `json:"current"`
	LastActive int64  `json:"lastActive"`
}

// personSignins is one account at a glance: how many devices are signed in, when it was last used, its last sign-in and the
// failed attempts under its name in the log.
type personSignins struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	Devices    int    `json:"devices"`
	LastActive int64  `json:"lastActive"`
	LastSignin int64  `json:"lastSignin"`
	Failed     int    `json:"failed"`
}

func peopleSignedIn(users []store.User, sessions []sessionRow, logs []store.Signin) []personSignins {
	out := make([]personSignins, 0, len(users))
	for _, u := range users {
		p := personSignins{Name: u.Name, Role: u.Role}
		for _, x := range sessions {
			if x.User == u.Name {
				p.Devices++
				p.LastActive = max(p.LastActive, x.LastActive)
			}
		}
		for _, l := range logs {
			switch {
			case !strings.EqualFold(l.Name, u.Name):
			case l.OK:
				p.LastSignin = max(p.LastSignin, l.TS)
			default:
				p.Failed++
			}
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].LastActive > out[j].LastActive })
	return out
}

// revokeSession signs one device out.
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "bad id")
		return
	}
	if err := s.core().DeleteSessionByID(id); err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, "signed out")
}
