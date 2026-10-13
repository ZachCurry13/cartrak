package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	tileFresh   = 30 * 24 * time.Hour // a cached tile is reused for this long
	tilePurge   = 60 * 24 * time.Hour // and deleted from disk after this long
	tileMaxSize = 1 << 20             // no real tile is near 1 MB
	maxZoom     = 19
)

// tileFlight makes concurrent requests for the same tile wait for one download.
type tileFlight struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
	slots chan struct{} // at most a few downloads from the tile server at once
}

func (f *tileFlight) lock(key string) *sync.Mutex {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locks == nil {
		f.locks = map[string]*sync.Mutex{}
		f.slots = make(chan struct{}, 4)
	}
	if f.locks[key] == nil {
		f.locks[key] = &sync.Mutex{}
	}
	return f.locks[key]
}

// mapConfig tells the page whether maps are on and what credit line to show.
func (s *Server) mapConfig(w http.ResponseWriter, r *http.Request) {
	on := s.mapsOn()
	out := map[string]any{"enabled": on}
	if on {
		out["tiles"] = "/api/tiles/{z}/{x}/{y}"
		out["attribution"] = s.Cfg.MapAttribution()
	}
	ok(w, out)
}

func (s *Server) mapsOn() bool { return s.Cfg != nil && s.TileDir != "" && s.Cfg.MapEnabled() }

// tile serves one map tile, downloading it from the tile server the first time and keeping a
// copy on disk. Only signed-in people can use it, so the server is not an open proxy; only
// raster image types are ever passed on, so the tile server cannot send us a page or script.
func (s *Server) tile(w http.ResponseWriter, r *http.Request) {
	if !s.mapsOn() {
		fail(w, 404, "maps are switched off")
		return
	}
	z, x, y, good := tileNumbers(r.PathValue("z"), r.PathValue("x"), r.PathValue("y"))
	if !good {
		fail(w, 400, "bad tile")
		return
	}
	path := filepath.Join(s.TileDir, strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y))
	key := fmt.Sprintf("%d/%d/%d", z, x, y)
	l := s.tiles.lock(key)
	l.Lock()
	defer l.Unlock()

	if body := readTile(path, tileFresh); body != nil {
		sendTile(w, body)
		return
	}
	body, err := s.downloadTile(z, x, y)
	if err != nil {
		if stale := readTile(path, 10*tilePurge); stale != nil { // an old copy beats a gap in the map
			sendTile(w, stale)
			return
		}
		log.Printf("map tile %s: %v", key, err)
		fail(w, 502, "map tiles are unavailable right now")
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		tmp := path + ".tmp"
		if os.WriteFile(tmp, body, 0o644) == nil {
			_ = os.Rename(tmp, path)
		}
	}
	sendTile(w, body)
}

// tileNumbers checks z/x/y are whole numbers in range (y may end in .png).
func tileNumbers(zs, xs, ys string) (z, x, y int, ok bool) {
	ys = strings.TrimSuffix(ys, ".png")
	var e1, e2, e3 error
	z, e1 = strconv.Atoi(zs)
	x, e2 = strconv.Atoi(xs)
	y, e3 = strconv.Atoi(ys)
	if e1 != nil || e2 != nil || e3 != nil || z < 0 || z > maxZoom {
		return 0, 0, 0, false
	}
	n := 1 << z
	return z, x, y, x >= 0 && x < n && y >= 0 && y < n
}

func readTile(path string, maxAge time.Duration) []byte {
	fi, err := os.Stat(path)
	if err != nil || time.Since(fi.ModTime()) > maxAge {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil || imageType(b) == "" {
		return nil
	}
	return b
}

// imageType names the raster type of the bytes, or "" for anything else (pages, SVG, scripts).
func imageType(b []byte) string {
	switch t := http.DetectContentType(b); t {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return t
	}
	return ""
}

func sendTile(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", imageType(b))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(b)
}

func (s *Server) downloadTile(z, x, y int) ([]byte, error) {
	rep := strings.NewReplacer("{z}", strconv.Itoa(z), "{x}", strconv.Itoa(x), "{y}", strconv.Itoa(y))
	req, err := http.NewRequest("GET", rep.Replace(s.Cfg.MapTileURL()), nil)
	if err != nil {
		return nil, err
	}
	// The tile servers' usage rules ask for an identifying User-Agent.
	req.Header.Set("User-Agent", "CarTrak/"+s.Version+" (+https://github.com/ZachCurry13/cartrak)")
	s.tiles.slots <- struct{}{}
	defer func() { <-s.tiles.slots }()
	c := s.TileClient
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tile server answered HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, tileMaxSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > tileMaxSize || imageType(b) == "" {
		return nil, fmt.Errorf("the tile server did not send a map image")
	}
	return b, nil
}

// purgeTiles deletes cached tiles nobody has needed for two months.
func (s *Server) purgeTiles(now time.Time) {
	if s.TileDir == "" {
		return
	}
	_ = filepath.Walk(s.TileDir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && now.Sub(fi.ModTime()) > tilePurge {
			_ = os.Remove(p)
		}
		return nil
	})
}
