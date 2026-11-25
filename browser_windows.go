//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const createNoWindow = 0x08000000

func findBrowser() string {
	pf86 := os.Getenv("ProgramFiles(x86)")
	pf := os.Getenv("ProgramFiles")
	la := os.Getenv("LOCALAPPDATA")
	for _, p := range []string{
		filepath.Join(pf86, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(la, `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(la, `Google\Chrome\Application\chrome.exe`),
		filepath.Join(pf, `BraveSoftware\Brave-Browser\Application\brave.exe`),
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// apre l'interfaccia in una finestra "app" senza barre del browser
func openAppWindow(url string) {
	if !getSettings().NativeFrame && openWebViewWindow(url) {
		return
	}
	openAppWindowExternal(url)
}

// finestra "app" di Edge (processo separato: resta aperta anche se questo processo termina)
func openAppWindowExternal(url string) {
	b := findBrowser()
	if b == "" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		return
	}
	prof := filepath.Join(dataDir, "window")
	cmd := exec.Command(b,
		"--app="+url,
		"--user-data-dir="+prof,
		"--window-size=1480,920",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-mode",
		"--disable-features=Translate,msEdgeStartupBoost,msImplicitSignin,msEdgeSidebarV2",
		"--disable-sync",
		"--disable-background-timer-throttling",
		"--disable-renderer-backgrounding",
		"--disable-backgrounding-occluded-windows",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	cmd.Start()
}

// riga di comando di un processo (per trovare porta e token del client LoL)
func processCommandLine(name string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-CimInstance Win32_Process -Filter \"name='"+name+"'\").CommandLine")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func setDPIAware() {
	user32 := syscall.NewLazyDLL("user32.dll")
	if p := user32.NewProc("SetProcessDpiAwarenessContext"); p.Find() == nil {
		p.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
		return
	}
	if p := user32.NewProc("SetProcessDPIAware"); p.Find() == nil {
		p.Call()
	}
}

func hiddenCmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return c
}

// apre un link nel browser predefinito
func openURL(u string) { exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start() }

// apre una cartella in Esplora file (selezionando il file, se indicato)
func openFolder(p string) {
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		exec.Command("explorer", "/select,", p).Start()
		return
	}
	exec.Command("explorer", p).Start()
}

// guida di un sito in una finestra a sé (Edge in modalità app, senza barre del browser)
func openGuideWindow(u string) {
	b := findBrowser()
	if b == "" {
		openURL(u)
		return
	}
	cmd := exec.Command(b, "--app="+u, "--user-data-dir="+filepath.Join(dataDir, "guide"), "--window-size=1400,900",
		"--no-first-run", "--no-default-browser-check", "--disable-features=Translate,msEdgeSidebarV2", "--disable-sync")
	cmd.Start()
}

// legge una pagina con Edge nascosto (headless) e restituisce l'HTML già caricato
var dumpMu sync.Mutex

func browserDump(u string) (string, error) {
	b := findBrowser()
	if b == "" {
		return "", fmt.Errorf("Edge non trovato")
	}
	dumpMu.Lock() // una pagina alla volta
	defer dumpMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, b, "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--mute-audio",
		"--user-data-dir="+filepath.Join(dataDir, "lettore"), "--virtual-time-budget=8000",
		"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
		"--dump-dom", u)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	out, err := cmd.Output()
	if err != nil && len(out) < 1000 {
		return "", err
	}
	return string(out), nil
}
