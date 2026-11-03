package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(Files, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// material.css holds the components and tokens.css the values: a raw radius, font size or colour in a component is a value the next
// design change would miss. See docs/DESIGN.md.
func TestComponentsTakeTheirValuesFromTheTokens(t *testing.T) {
	css := read(t, "static/css/material.css")
	raw := regexp.MustCompile(`border-radius:[0-9.]+px|font-size:[0-9.]+(px|rem|em)|#[0-9a-fA-F]{3,6}\b|cubic-bezier\(`)
	// the registered ripple property, the theme-aware nav rules and the switch check mark are the only places a literal is expected
	for i, line := range strings.Split(css, "\n") {
		if strings.Contains(line, "data:image/svg+xml") {
			continue
		}
		if m := raw.FindString(line); m != "" && !strings.HasPrefix(strings.TrimSpace(line), "/*") {
			t.Errorf("material.css line %d uses %q: add or use a token in tokens.css\n%s", i+1, m, line)
		}
	}
}

// A component that names a token nobody defines silently loses its style. The tokens are defined in tokens.css and the theme
// colours in look.css; the ripple's position is set by touchfeedback.js and its size is registered in material.css.
func TestEveryTokenAComponentUsesIsDefined(t *testing.T) {
	defs := read(t, "static/css/tokens.css") + read(t, "static/css/look.css") + read(t, "static/css/material.css")
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(--[a-z0-9-]+)\s*:`).FindAllStringSubmatch(defs, -1) {
		defined[m[1]] = true
	}
	for _, v := range []string{"--rx", "--ry", "--rr"} {
		defined[v] = true
	}
	for _, m := range regexp.MustCompile(`var\((--[a-z0-9-]+)`).FindAllStringSubmatch(read(t, "static/css/material.css"), -1) {
		if !defined[m[1]] {
			t.Errorf("material.css uses %s, which is defined nowhere", m[1])
		}
	}
	if strings.Contains(read(t, "static/css/material.css"), "\n  --r-") {
		t.Error("material.css defines a token: tokens belong in tokens.css")
	}
}

// Every stylesheet and script the app loads must be in the service worker's list, or it is missing offline and after an update.
// The two scripts of the standalone pages (Ford's sign-in return and the share page) are loaded from their own HTML and stay uncached.
func TestEveryStylesheetAndScriptIsCachedByTheServiceWorker(t *testing.T) {
	sw := read(t, "static/sw.js")
	err := fs.WalkDir(Files, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := strings.TrimPrefix(p, "static")
		if (strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".js")) && name != "/sw.js" && name != "/js/callback.js" && name != "/js/share.js" && !strings.Contains(sw, "'"+name+"'") {
			t.Errorf("%s is not in the SHELL list in sw.js", name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// An icon the manifest, the page or the service worker names must exist, or the app installs with a broken icon. They live in a
// versioned folder (icons/v2): change the art, change the folder, and no cache can keep serving the old picture.
func TestEveryIconTheAppNamesExists(t *testing.T) {
	re := regexp.MustCompile(`/icons/v[0-9]+/[A-Za-z0-9._-]+`)
	for _, f := range []string{"static/manifest.webmanifest", "static/index.html", "static/sw.js"} {
		found := re.FindAllString(read(t, f), -1)
		if len(found) == 0 {
			t.Errorf("%s names no icons in a versioned folder", f)
		}
		for _, p := range found {
			if _, err := fs.Stat(Files, "static"+p); err != nil {
				t.Errorf("%s names %s, which does not exist", f, p)
			}
		}
	}
	if strings.Contains(read(t, "static/manifest.webmanifest"), "\"/icons/icon") {
		t.Error("the manifest still points at an unversioned icon address")
	}
}
