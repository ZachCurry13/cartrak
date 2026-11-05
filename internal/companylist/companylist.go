// Package companylist reads a pasted list of company names (one per line, often with an account number and a country on the
// end) and matches each one against places already known.
package companylist

import (
	"regexp"
	"strings"
)

// Entry is one line of the list.
type Entry struct {
	Raw  string `json:"raw"`
	Name string `json:"name"` // the company, with the number and country taken off
	Ref  string `json:"ref"`  // the number from the line, which is how a company is remembered; "" when the line has none
}

// MaxLines is how many companies one list may have: the route planner takes ten stops.
const MaxLines = 10

var tail = regexp.MustCompile(`^(.*?)\s*[-–—]\s*(\d{3,})\s*(?:[-–—]\s*[A-Za-z][A-Za-z .]{1,30})?$`)

// activity is a line a customer system puts under each company: the days since the last contact and how it was made. It names no
// company, so it is not one.
var activity = regexp.MustCompile(`^\d{1,4}\s*[-–—:]\s*[A-Za-z][A-Za-z /&]{1,40}$`)

// Parse splits pasted text into entries, dropping blank lines and repeats. more is how many were left out past MaxLines.
func Parse(text string) (out []Entry, more int) {
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-*•· "))
		if line == "" || (activity.MatchString(line) && tail.FindStringSubmatch(line) == nil) {
			continue
		}
		e := Entry{Raw: line, Name: line}
		if m := tail.FindStringSubmatch(line); m != nil && strings.TrimSpace(m[1]) != "" {
			e.Name, e.Ref = strings.TrimSpace(m[1]), m[2]
		}
		key := strings.ToLower(e.Ref + "|" + e.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(out) >= MaxLines {
			more++
			continue
		}
		out = append(out, e)
	}
	return out, more
}

var noise = map[string]bool{"inc": true, "corp": true, "corporation": true, "co": true, "company": true, "llc": true, "ltd": true, "limited": true, "the": true, "and": true, "of": true, "usa": true}

// Words reduces a name to its meaningful lower-case words: no punctuation, no "Inc" or "LLC".
func Words(s string) []string {
	var w []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') }) {
		if !noise[f] {
			w = append(w, f)
		}
	}
	return w
}

// Same says whether two names mean the same company: the same words, or one name's words all inside the other's (and at least four letters
// long, so a short word does not match everything).
func Same(a, b string) bool {
	x, y := Words(a), Words(b)
	if len(x) == 0 || len(y) == 0 {
		return false
	}
	if len(x) > len(y) {
		x, y = y, x
	}
	in := map[string]bool{}
	for _, w := range y {
		in[w] = true
	}
	for _, w := range x {
		if !in[w] {
			return false
		}
	}
	return len(strings.Join(x, "")) >= 4
}
