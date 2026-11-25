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
	go remoteRefresh() // le build appena caricate compaiono subito
	return nil
}

var collectState struct {
	sync.Mutex
	Running bool
	Step    string
	Done    int
	Total   int
	At      int64
	Start   time.Time
	Got     int // pagine lette con successo nell'ultima raccolta
}

func collectorInfo() any {
	collectState.Lock()
	defer collectState.Unlock()
	eta := 0
	if collectState.Running && collectState.Done > 0 && collectState.Total > collectState.Done {
		per := time.Since(collectState.Start).Seconds() / float64(collectState.Done)
		eta = int(per * float64(collectState.Total-collectState.Done))
	}
	return map[string]any{"enabled": isCollector(), "running": collectState.Running, "step": collectState.Step, "done": collectState.Done,
		"total": collectState.Total, "at": collectState.At, "eta": eta}
}

func setStep(step string, done, total int) {
	collectState.Lock()
	collectState.Step = step
	if done >= 0 {
		collectState.Done = done
	}
	if total >= 0 {
		collectState.Total = total
	}
	collectState.Unlock()
}

func stepDone(ok bool) {
	collectState.Lock()
	collectState.Done++
	if ok {
		collectState.Got++
	}
	collectState.Unlock()
}

func collectLoop() {
	var last struct {
		At      int64  `json:"at"`
		Version string `json:"version"`
	}
	readJSON(cacheFile("raccolta.json"), &last)
	if last.Version != appVersion { // nuova versione dell'app: raccoglie di nuovo subito
		last.At = 0
	}
	collectState.Lock()
	collectState.At = last.At
	collectState.Unlock()
	time.Sleep(15 * time.Second)
	retry := time.Time{}
	for {
		collectState.Lock()
		at := collectState.At
		collectState.Unlock()
		if isCollector() && time.Since(time.Unix(at, 0)) > 6*time.Hour && time.Now().After(retry) && len(seasonsAvailable()) > 0 {
			collect()
			collectState.Lock()
			got := collectState.Got
			collectState.Unlock()
			if got > 0 {
				now := time.Now().Unix()
				collectState.Lock()
				collectState.At = now
				collectState.Unlock()
				writeJSON(cacheFile("raccolta.json"), map[string]any{"at": now, "version": appVersion})
			} else {
				retry = time.Now().Add(30 * time.Minute) // non è riuscita: riprova tra mezz'ora
			}
		}
		time.Sleep(time.Minute)
	}
}

func collect() {
	collectState.Lock()
	if collectState.Running {
		collectState.Unlock()
		return
	}
	collectState.Running, collectState.Done, collectState.Total, collectState.Got = true, 0, 0, 0
	collectState.Start = time.Now()
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

	// quante pagine in tutto (per la barra e il tempo stimato)
	d4bURLs := map[int][]string{}
	nD4b := 0
	for n := range avail {
		v := seasonView(n)
		if v == nil {
			continue
		}
		seen := map[string]bool{}
		for _, b := range v.Builds {
			if b.Src == "D4Builds" && strings.Contains(b.URL, "d4builds.gg/builds/") && !seen[keyURL(b.URL)] {
				seen[keyURL(b.URL)] = true
				d4bURLs[n] = append(d4bURLs[n], b.URL)
				nD4b++
			}
		}
	}
	setStep("Mobalytics: elenco delle build", 0, nD4b+10)

	// 1) Mobalytics: elenchi e guida dei boss
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
	nMoba := 0
	for _, l := range bySeason {
		nMoba += len(l)
	}
	setStep("Mobalytics: dettagli delle build", 10, 10+nMoba+nD4b)
	for n, list := range bySeason {
		details := map[string]*PBuild{}
		for _, b := range list {
			setStep(fmt.Sprintf("Mobalytics · stagione %d", n), -1, -1)
			mobaDetailCache.Lock()
			delete(mobaDetailCache.m, b.URL)
			mobaDetailCache.Unlock()
			d, err := mobaBuildDetail(b.URL)
			if err == nil {
				details[keyURL(b.URL)] = d
			}
			stepDone(err == nil)
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
	for n, urls := range d4bURLs {
		details := map[string]*PBuild{}
		for _, u := range urls {
			setStep(fmt.Sprintf("D4Builds · stagione %d", n), -1, -1)
			var d *PBuild
			t0 := time.Now()
			f, err := d4bFullRead(u)
			if err == nil && time.Since(t0) > 2*time.Second {
				log.Printf("raccolta: d4builds %s letta in %ds", strings.TrimPrefix(keyURL(u), "https://d4builds.gg/builds/"), int(time.Since(t0).Seconds()))
			}
			if err == nil {
				d = f.slim()
				if e := uploadFull(u, f); e != nil {
					log.Printf("raccolta: d4builds completa %s: %v", u, e)
				}
				details[keyURL(u)] = d
			} else {
				log.Printf("raccolta: d4builds %s: %v", u, err)
			}
			stepDone(err == nil)
			if len(details) >= 10 {
				if err := uploadData("d4bDetails", n, details); err != nil {
					log.Printf("raccolta: d4bDetails S%d: %v", n, err)
				}
				details = map[string]*PBuild{}
			}
			time.Sleep(500 * time.Millisecond)
		}
		if len(details) > 0 {
			uploadData("d4bDetails", n, details)
		}
	}
	if len(mc.Builds) > 0 {
		stepDone(true)
	}
	collectState.Lock()
	log.Printf("raccolta: finita, %d pagine lette su %d", collectState.Got, collectState.Total)
	collectState.Unlock()
	go remoteRefresh()
}

func uploadFull(u string, f *D4Full) error {
	base, tok := remoteBase()
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	q := fmt.Sprintf("%sapp.php?upload=full&id=%s&k=%s", base, hashURL(keyURL(u)), url.QueryEscape(tok))
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Post(q, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	os.WriteFile(cacheFile("full_"+hashURL(keyURL(u))+".json"), b, 0o644)
	return nil
}

// build completa dal server, tenuta in cache per 3 ore (e usata anche se il server non risponde)
func serverFull(u string) []byte {
	id := hashURL(keyURL(u))
	p := cacheFile("full_" + id + ".json")
	if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 3*time.Hour {
		if b, err := os.ReadFile(p); err == nil && bytes.Contains(b, []byte(fmt.Sprintf(`"xv":%d`, d4bExtractVer))) {
			return b
		}
	}
	b, err := remoteGet("what=full&id=" + id)
	if err == nil && len(b) > 2 && b[0] == '{' {
		os.WriteFile(p, b, 0o644)
		return b
	}
	if b, err := os.ReadFile(p); err == nil {
		return b
	}
	return nil
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
