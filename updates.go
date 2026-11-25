package main

// Distribuzione agli amici: l'app di Max pubblica sul server ogni nuova versione,
// le app di tutti la scaricano da sole e si aggiornano (come BetterStomp).

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
	"time"
)

type pubVersion struct {
	Version string `json:"version"`
	Size    int64  `json:"size"`
	Sha     string `json:"sha256"`
}

func publishedVersion() (pubVersion, error) {
	var v pubVersion
	base, _ := remoteBase()
	if base == "" {
		return v, fmt.Errorf("server non configurato")
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Get(base + "app.php?v=1")
	if err != nil {
		return v, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&v)
	return v, err
}

// solo sul PC di Max (ha il codice del server e la cartella dei file del server)
func canPublish() bool {
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

func publishExe() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	sha := hex.EncodeToString(sum[:])
	base, tok := remoteBase()
	const piece = 1 << 20
	n := (len(b) + piece - 1) / piece
	cl := &http.Client{Timeout: 2 * time.Minute}
	for i := 0; i < n; i++ {
		end := (i + 1) * piece
		if end > len(b) {
			end = len(b)
		}
		q := fmt.Sprintf("%sapp.php?k=%s&up=1&i=%d&n=%d&ver=%s&sha=%s", base, url.QueryEscape(tok), i, n, appVersion, sha)
		resp, err := cl.Post(q, "application/octet-stream", bytes.NewReader(b[i*piece:end]))
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("pezzo %d: HTTP %d %s", i, resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}
	return nil
}

func downloadUpdate(v pubVersion) error {
	base, _ := remoteBase()
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Get(base + "app.php?get=1")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if v.Sha != "" && hex.EncodeToString(sum[:]) != v.Sha {
		return fmt.Errorf("file scaricato rovinato")
	}
	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, "Hellrise.update.exe.part")
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "Hellrise.update.exe"))
}

func serverUpdateLoop() {
	time.Sleep(30 * time.Second)
	for {
		if appVersion != "0" {
			v, err := publishedVersion()
			switch {
			case err != nil:
			case canPublish() && v.Version < appVersion:
				if err := publishExe(); err != nil {
					log.Printf("pubblicazione versione %s: %v", appVersion, err)
				} else {
					log.Printf("versione %s pubblicata sul server per gli amici", appVersion)
				}
			case v.Version > appVersion && !canPublish():
				exe, _ := os.Executable()
				if !fileExists(filepath.Join(filepath.Dir(exe), "Hellrise.update.exe")) {
					if err := downloadUpdate(v); err != nil {
						log.Printf("aggiornamento %s: %v", v.Version, err)
					} else {
						log.Printf("scaricata la versione %s: si installa alla chiusura o con «Aggiorna ora»", v.Version)
					}
				}
			}
		}
		time.Sleep(10 * time.Minute)
	}
}
