package main

// Mappa interattiva: disegni e punti di interesse di d4builds.gg/map (tessere sunderarmor.com),
// salvati sul PC man mano che servono. "Trovati" (altari, dungeon…) restano solo in locale.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mapHC = &http.Client{Timeout: 30 * time.Second}

func mapGet(u, ref string) ([]byte, error) {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/131.0 Safari/537.36")
	if ref != "" {
		req.Header.Set("Referer", ref)
	}
	resp, err := mapHC.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return b, nil
}

func mapMarkers() ([]byte, error) {
	p := cacheFile("d4b_map.json")
	if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 7*24*time.Hour {
		return os.ReadFile(p)
	}
	b, err := mapGet("https://d4builds.gg/page-data/map/page-data.json", "https://d4builds.gg/map/")
	if err != nil {
		if old, e2 := os.ReadFile(p); e2 == nil {
			return old, nil
		}
		return nil, err
	}
	os.WriteFile(p, b, 0o644)
	return b, nil
}

var tileMu sync.Mutex

func serveTile(w http.ResponseWriter, r *http.Request) {
	// /api/tile/z/x/y
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/tile/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	z, e1 := strconv.Atoi(parts[0])
	x, e2 := strconv.Atoi(parts[1])
	y, e3 := strconv.Atoi(strings.TrimSuffix(parts[2], ".jpg"))
	if e1 != nil || e2 != nil || e3 != nil || z < 0 || z > 6 || x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		http.NotFound(w, r)
		return
	}
	dir := filepath.Join(dataDir, "cache", "tiles", strconv.Itoa(z), strconv.Itoa(x))
	p := filepath.Join(dir, strconv.Itoa(y)+".jpg")
	b, err := os.ReadFile(p)
	if err != nil {
		b, err = mapGet(fmt.Sprintf("https://sunderarmor.com/DIABLO4/VoHMap/%d/%d/%d.jpg", z, x, y), "https://d4builds.gg/")
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		tileMu.Lock()
		os.MkdirAll(dir, 0o755)
		os.WriteFile(p, b, 0o644)
		tileMu.Unlock()
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Write(b)
}

func mapFoundPath() string { return filepath.Join(dataDir, "mappa_trovati.json") }
