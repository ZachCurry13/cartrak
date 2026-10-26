package server

import (
	"regexp"
	"strings"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// fileSlug is a name written so it is safe in a file name: "My Mach-E" becomes "my-mach-e", and nothing becomes "car".
func fileSlug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "car"
	}
	if len(s) > 30 {
		s = strings.Trim(s[:30], "-")
	}
	return s
}
