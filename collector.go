package main

// Raccolta "dal PC" per i siti che bloccano il server (D4Builds si legge solo con un browser,
// Mobalytics risponde 403): solo l'app di Max (quella con la cartella "Server per aversano.design")
// legge le pagine in sottofondo con Edge nascosto e carica i risultati sul server.
// Tutte le app poi li ricevono già pronti dal server, senza caricamenti.

import (
	"bytes"
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

func keyURL(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	return strings.TrimSuffix(u, "/")
}

func isCollector() bool {
	base, tok := remoteBase()
	if tok == "" || !strings.Contains(base, "aversano.design") {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return fileExists(filepath.Join(filepath.Dir(exe), "Server per aversano.design", "hr-data"))
}

func uploadData(key string, season int, v any) error {
	base, tok := remoteBase()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("%sapp.php?upload=%s&season=%d&k=%s", base, key, season, url.QueryEscape(tok))
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Post(u, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

var collectState struct {
	sync.Mutex
	Running bool   `json:"running"`
	Step    string `json:"step"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	At      int64  `json:"at"`
}

func collectorInfo() any {
	collectState.Lock()
	defer collectState.Unlock()
	return map[string]any{"enabled": isCollector(), "running": collectState.Running, "step": collectState.Step, "done": collectState.Done, "total": collectState.Total, "at": collectState.At}
}

func setStep(step string, done, total int) {
	collectState.Lock()
	collectState.Step, collectState.Done, collectState.Total = step, done, total
	collectState.Unlock()
}

func collectLoop() {
	var last struct {
		At int64 `json:"at"`
	}
	readJSON(cacheFile("raccolta.json"), &last)
	collectState.Lock()
	collectState.At = last.At
	collectState.Unlock()
	time.Sleep(40 * time.Second)
	for {
		collectState.Lock()
		at := collectState.At
		collectState.Unlock()
		if isCollector() && time.Since(time.Unix(at, 0)) > 6*time.Hour && len(seasonsAvailable()) > 0 {
			collect()
			now := time.Now().Unix()
			collectState.Lock()
			collectState.At = now
			collectState.Unlock()
			writeJSON(cacheFile("raccolta.json"), map[string]int64{"at": now})
		}
		time.Sleep(10 * time.Minute)
	}
}

func collect() {
	collectState.Lock()
	if collectState.Running {
		collectState.Unlock()
		return
	}
	collectState.Running = true
	collectState.Unlock()
	defer func() {
		collectState.Lock()
		collectState.Running, collectState.Step = false, ""
		collectState.Unlock()
	}()
	live := remoteLive()
	avail := map[int]bool{}
	for _, s := range seasonsAvailable() {
		avail[s.N] = true
	}
	log.Printf("raccolta: inizio (stagione live %d)", live)

	// 1) Mobalytics: elenchi e guida dei boss
	setStep("Mobalytics: elenco build", 0, 0)
	mobaRefresh()
	mc := mobaSnapshot()
	bySeason := map[int][]mobaBuild{}
	for _, b := range mc.Builds {
		n := b.Season
		if n == 0 {
			n = live
		}
		if avail[n] {
			bySeason[n] = append(bySeason[n], b)
		}
	}
	for n, list := range bySeason {
		if err := uploadData("mobalytics", n, list); err != nil {
			log.Printf("raccolta: mobalytics S%d: %v", n, err)
		}
	}
	if len(mc.Bosses) > 0 && live > 0 {
		info := map[string]any{"source": "mobalytics.gg", "url": "https://mobalytics.gg/diablo-4/guides/diablo-4-boss-loot-tables", "bosses": mc.Bosses}
		if err := uploadData("bossInfo", live, info); err != nil {
			log.Printf("raccolta: boss: %v", err)
		}
	}

	// 2) dettagli delle build Mobalytics
	for n, list := range bySeason {
		details := map[string]*PBuild{}
		for i, b := range list {
			setStep(fmt.Sprintf("Mobalytics S%d: dettagli build", n), i, len(list))
			mobaDetailCache.Lock()
			delete(mobaDetailCache.m, b.URL)
			mobaDetailCache.Unlock()
			if d, err := mobaBuildDetail(b.URL); err == nil {
				details[keyURL(b.URL)] = d
			}
			if len(details) >= 10 {
				if err := uploadData("mobaDetails", n, details); err != nil {
					log.Printf("raccolta: mobaDetails S%d: %v", n, err)
				}
				details = map[string]*PBuild{}
			}
			time.Sleep(1500 * time.Millisecond)
		}
		if len(details) > 0 {
			uploadData("mobaDetails", n, details)
		}
	}

	// 3) dettagli delle build D4Builds (elenco del sito e build di Rob)
	for n := range avail {
		v := seasonView(n)
		if v == nil {
			continue
		}
		seen := map[string]bool{}
		var urls []string
		for _, b := range v.Builds {
			if b.Src == "D4Builds" && strings.Contains(b.URL, "d4builds.gg/builds/") && !seen[keyURL(b.URL)] {
				seen[keyURL(b.URL)] = true
				urls = append(urls, b.URL)
			}
		}
		details := map[string]*PBuild{}
		for i, u := range urls {
			setStep(fmt.Sprintf("D4Builds S%d: lettura build", n), i, len(urls))
			if d, err := d4bBuildDetail(u); err == nil {
				details[keyURL(u)] = d
			} else {
				log.Printf("raccolta: d4builds %s: %v", u, err)
			}
			if len(details) >= 10 {
				if err := uploadData("d4bDetails", n, details); err != nil {
					log.Printf("raccolta: d4bDetails S%d: %v", n, err)
				}
				details = map[string]*PBuild{}
			}
			time.Sleep(1500 * time.Millisecond)
		}
		if len(details) > 0 {
			uploadData("d4bDetails", n, details)
		}
	}
	log.Printf("raccolta: finita")
	go remoteRefresh()
}

// dettaglio già raccolto sul server (in una delle stagioni)
func serverDetail(kind, u string) *PBuild {
	k := keyURL(u)
	for _, s := range seasonsAvailable() {
		v := seasonView(s.N)
		if v == nil {
			continue
		}
		m := v.D4bDetails
		if kind == "moba" {
			m = v.MobaDetails
		}
		if b := m[k]; b != nil {
			return b
		}
	}
	return nil
}
