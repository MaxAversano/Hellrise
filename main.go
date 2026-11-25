package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync/atomic"
	"time"
)

//go:embed web
var webFS embed.FS

var (
	appURL     string
	appVersion = "0"
	dataDir    string
	lastPing   atomic.Int64
)

func initDataDir() {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dataDir = filepath.Join(filepath.Dir(exe), "Hellrise-data")
	os.MkdirAll(dataDir, 0o755)
	for _, d := range []string{"cache", "planner", "server"} {
		os.MkdirAll(filepath.Join(dataDir, d), 0o755)
	}
}

// cerca una porta libera; se su una porta risponde già Hellrise riapre solo la sua finestra
func listenFree() net.Listener {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	for p := 47911; p < 47950; p++ {
		addr := fmt.Sprintf("127.0.0.1:%d", p)
		if ln, err := net.Listen("tcp", addr); err == nil {
			appURL = "http://" + addr + "/"
			return ln
		}
		if resp, err := client.Get("http://" + addr + "/api/whoami"); err == nil {
			var who struct {
				App string `json:"app"`
			}
			json.NewDecoder(resp.Body).Decode(&who)
			resp.Body.Close()
			if who.App == "Hellrise" {
				if r2, err := client.Get("http://" + addr + "/api/win?a=open"); err == nil {
					r2.Body.Close()
				}
				os.Exit(0)
			}
		}
	}
	return nil
}

func main() {
	initDataDir()
	ln := listenFree()
	if ln == nil {
		return
	}
	if f, err := os.OpenFile(filepath.Join(dataDir, "log.txt"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(f)
	}
	if f, err := os.OpenFile(filepath.Join(dataDir, "crash.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		debug.SetCrashOutput(f, debug.CrashOptions{})
	}
	log.Printf("Hellrise %s su %s", appVersion, appURL)
	loadSettings()
	setDPIAware()

	mux := http.NewServeMux()
	setupAPI(mux)
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache") // dopo un aggiornamento la finestra usa subito i file nuovi
		files.ServeHTTP(w, r)
	})
	go http.Serve(ln, mux)

	go startTrayThread()
	go remoteLoop()
	go gameDataLoop()
	go mobaLoad()
	go collectLoop()
	go eventNotifyLoop()
	go localUpdateLoop()
	go serverSyncLoop()
	go cleanupOldExe()
	go framelessLoop()
	go startOverlay()

	start := time.Now()
	hidden := false
	for _, a := range os.Args[1:] {
		hidden = hidden || a == "--nascosto"
	}
	if !hidden {
		go openAppWindow(appURL)
	}
	// resta aperto finché la finestra è aperta (o sempre, con l'avvio automatico per le notifiche dei boss)
	for {
		time.Sleep(3 * time.Second)
		lp := lastPing.Load()
		idle := lp == 0 && time.Since(start) > 3*time.Minute || lp != 0 && time.Since(time.Unix(lp, 0)) > 20*time.Second
		if !idle || getSettings().Autostart || getSettings().KeepInTray {
			continue
		}
		if collectorInfo().(map[string]any)["running"] == true { // finisce la raccolta in corso
			continue
		}
		if getSettings().Overlay.Enabled && diabloRunning() { // l'overlay resta attivo finché giochi
			continue
		}
		log.Printf("chiusura: finestra chiusa")
		appExit()
	}
}

func restartApp() {
	time.Sleep(300 * time.Millisecond)
	if exe, err := os.Executable(); err == nil {
		relaunchLater(exe)
	}
	appExit()
}

// aggiornamento senza chiudere a mano: l'exe in uso si può rinominare, non sovrascrivere
func selfUpdate() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	upd := filepath.Join(filepath.Dir(exe), "Hellrise.update.exe")
	if _, err := os.Stat(upd); err != nil {
		return err
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(upd, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	return nil
}

func cleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if upd := filepath.Join(filepath.Dir(exe), "Hellrise.update.exe"); fileExists(upd) {
		if selfUpdate() == nil {
			restartApp()
			return
		}
		os.Remove(upd)
	}
	for i := 0; i < 10; i++ {
		if os.Remove(exe+".old") == nil || !fileExists(exe+".old") {
			return
		}
		time.Sleep(time.Second)
	}
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
