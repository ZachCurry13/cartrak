package tunnel

import (
	"errors"
	"regexp"
	"strings"
)

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
