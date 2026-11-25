//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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
