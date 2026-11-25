package main

// Dati dal server aversano.design (hr-data): elenco stagioni, build, tier list, loot e orari dei boss.
// Tutto viene salvato anche sul PC, così l'app funziona subito all'avvio e anche senza connessione.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SeasonInfo struct {
	N       int              `json:"n"`
	Updated map[string]int64 `json:"updated"`
	PTR     bool             `json:"ptr"`
	Live    bool             `json:"live"`
}

type RemoteIndex struct {
	Live      int               `json:"live"`
	Seasons   []SeasonInfo      `json:"seasons"`
	LastRun   int64             `json:"lastRun"`
	Runs      int               `json:"runs"`
	Errors    map[string]string `json:"errors"`
	Events    json.RawMessage   `json:"events"`
	PatchesAt int64             `json:"patchesAt"`
}

var remote struct {
	sync.Mutex
	index   RemoteIndex
	status  string // "" ok, altrimenti il problema
	at      time.Time
	seasons map[int]json.RawMessage
}

var httpc = &http.Client{Timeout: 40 * time.Second}

func remoteBase() (string, string) {
	s := getSettings()
	u := strings.TrimSpace(s.RemoteURL)
	if u != "" && !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u, s.RemoteToken
}

func remoteGet(q string) ([]byte, error) {
	base, tok := remoteBase()
	if base == "" || tok == "" {
		return nil, fmt.Errorf("server non configurato")
	}
	req, _ := http.NewRequest("GET", base+"data.php?k="+url.QueryEscape(tok)+"&"+q, nil)
	req.Header.Set("User-Agent", "Hellrise/"+appVersion)
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		msg := strings.TrimSpace(string(b))
		if len(msg) > 120 {
			msg = msg[:120]
		}
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, msg)
	}
	return b, nil
}

func serverCache(name string) string { return filepath.Join(dataDir, "server", name) }

func loadRemoteCache() {
	remote.Lock()
	defer remote.Unlock()
	if remote.seasons == nil {
		remote.seasons = map[int]json.RawMessage{}
	}
	readJSON(serverCache("index.json"), &remote.index)
	for _, s := range remote.index.Seasons {
		if b, err := os.ReadFile(serverCache(fmt.Sprintf("s%d.json", s.N))); err == nil {
			remote.seasons[s.N] = b
		}
	}
}

func remoteRefresh() error {
	b, err := remoteGet("what=index")
	if err != nil {
		remote.Lock()
		remote.status = err.Error()
		remote.Unlock()
		return err
	}
	var idx RemoteIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return err
	}
	remote.Lock()
	old := remote.index
	remote.Unlock()
	keep := map[int]bool{}
	for _, s := range idx.Seasons {
		keep[s.N] = true
		changed := !fileExists(serverCache(fmt.Sprintf("s%d.json", s.N)))
		for _, o := range old.Seasons {
			if o.N == s.N {
				for k, t := range s.Updated {
					if o.Updated[k] != t {
						changed = true
					}
				}
			}
		}
		if !changed {
			continue
		}
		sb, err := remoteGet(fmt.Sprintf("what=season&n=%d", s.N))
		if err != nil {
			log.Printf("server: stagione %d non scaricata: %v", s.N, err)
			continue
		}
		os.WriteFile(serverCache(fmt.Sprintf("s%d.json", s.N)), sb, 0o644)
		remote.Lock()
		remote.seasons[s.N] = sb
		remote.Unlock()
		invalidateSeason(s.N)
		log.Printf("server: stagione %d aggiornata (%d KB)", s.N, len(sb)/1024)
	}
	// solo le due stagioni tenute dal server
	files, _ := filepath.Glob(serverCache("s*.json"))
	for _, f := range files {
		var n int
		if _, err := fmt.Sscanf(filepath.Base(f), "s%d.json", &n); err == nil && !keep[n] {
			os.Remove(f)
			remote.Lock()
			delete(remote.seasons, n)
			remote.Unlock()
		}
	}
	if idx.PatchesAt != old.PatchesAt || !fileExists(serverCache("patches.json")) {
		if pb, err := remoteGet("what=patches"); err == nil && len(pb) > 2 {
			os.WriteFile(serverCache("patches.json"), pb, 0o644)
		}
	}
	writeJSON(serverCache("index.json"), idx)
	remote.Lock()
	remote.index, remote.status, remote.at = idx, "", time.Now()
	remote.Unlock()
	updateWorldBoss(idx.Events)
	return nil
}

func remoteLoop() {
	loadRemoteCache()
	remote.Lock()
	ev := remote.index.Events
	remote.Unlock()
	updateWorldBoss(ev)
	for {
		if err := remoteRefresh(); err != nil {
			log.Printf("server: %v", err)
			time.Sleep(2 * time.Minute)
			continue
		}
		time.Sleep(8 * time.Minute)
	}
}

func seasonBytes(n int) json.RawMessage {
	remote.Lock()
	defer remote.Unlock()
	return remote.seasons[n]
}

// stagioni disponibili, dalla più recente; la prima è quella mostrata se l'utente non ne sceglie una
func seasonsAvailable() []SeasonInfo {
	remote.Lock()
	defer remote.Unlock()
	out := []SeasonInfo{}
	for _, s := range remote.index.Seasons {
		if remote.seasons[s.N] != nil {
			out = append(out, s)
		}
	}
	return out
}

func currentSeason(req int) int {
	av := seasonsAvailable()
	for _, s := range av {
		if s.N == req {
			return req
		}
	}
	if s := getSettings().Season; s > 0 {
		for _, x := range av {
			if x.N == s {
				return s
			}
		}
	}
	if len(av) > 0 {
		return av[0].N
	}
	return 0
}

func remoteStatus() map[string]any {
	remote.Lock()
	defer remote.Unlock()
	base, tok := remoteBase()
	return map[string]any{"configured": base != "" && tok != "", "error": remote.status, "at": remote.at.Unix(),
		"live": remote.index.Live, "lastRun": remote.index.LastRun, "errors": remote.index.Errors}
}

// ---------- file PHP del server: vengono caricati quando cambiano ----------

var serverFiles = []string{"lib.php", "run.php", "data.php", "ping.php", "index.php", "app.php"}

func serverSyncLoop() {
	time.Sleep(20 * time.Second)
	for {
		syncServerFiles()
		time.Sleep(time.Minute)
	}
}

func syncServerFiles() {
	base, tok := remoteBase()
	if tok == "" || !strings.Contains(base, "aversano.design") {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Join(filepath.Dir(exe), "Server per aversano.design", "hr-data")
	if !fileExists(dir) {
		return
	}
	statePath := filepath.Join(dataDir, "server_sync.json")
	done := map[string]string{}
	readJSON(statePath, &done)
	changed := false
	hc := &http.Client{Timeout: time.Minute}
	for _, name := range serverFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(data) < 10 {
			continue
		}
		sum := sha256.Sum256(data)
		h := hex.EncodeToString(sum[:])
		if done[name] == h {
			continue
		}
		// prima il server di Hellrise; la prima volta (cartella ancora vuota) passa da quello di BetterStomp
		targets := []string{base + "app.php?put=" + name, strings.TrimSuffix(base, "hr-data/") + "bs-data/app.php?put=hr/" + name}
		ok := false
		for _, t := range targets {
			resp, err := hc.Post(t+"&k="+url.QueryEscape(tok), "application/octet-stream", bytes.NewReader(data))
			if err != nil {
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.TrimSpace(string(body)) == "ok" {
				ok = true
				break
			}
			log.Printf("server: %s → HTTP %d %s", t, resp.StatusCode, strings.TrimSpace(string(body)))
		}
		if !ok {
			continue
		}
		log.Printf("server: %s aggiornato", name)
		done[name] = h
		changed = true
	}
	if changed {
		writeJSON(statePath, done)
	}
}

// ---------- aggiornamento lasciato nella cartella (Hellrise.update.exe) ----------

var updReady atomic_bool

type atomic_bool struct {
	sync.Mutex
	v bool
}

func (a *atomic_bool) Load() bool   { a.Lock(); defer a.Unlock(); return a.v }
func (a *atomic_bool) Store(v bool) { a.Lock(); a.v = v; a.Unlock() }

func localUpdateLoop() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	upd := filepath.Join(filepath.Dir(exe), "Hellrise.update.exe")
	for {
		time.Sleep(20 * time.Second)
		if !fileExists(upd) {
			updReady.Store(false)
			continue
		}
		updReady.Store(true)
		lp := lastPing.Load()
		if lp == 0 || time.Since(time.Unix(lp, 0)) > 60*time.Second { // finestra chiusa: si aggiorna da sola
			log.Printf("aggiornamento in sottofondo: riavvio")
			if selfUpdate() == nil {
				restartApp()
				return
			}
		}
	}
}
