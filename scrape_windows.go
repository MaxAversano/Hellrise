//go:build windows

package main

// Lettore di pagine: un WebView2 (lo stesso motore di Edge usato per la finestra dell'app) aperto
// in una finestra nascosta fuori dallo schermo. Carica la pagina pubblica come un visitatore qualsiasi,
// aspetta che sia pronta e ne restituisce l'HTML. Serve per D4Builds (contenuto disegnato dal sito)
// e Mobalytics (blocca i programmi che non sono un browser).

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type scrapeJob struct {
	url     string
	ready   string // espressione JavaScript: vera quando la pagina è pronta
	start   string // script da avviare a pagina pronta (facoltativo)
	poll    string // espressione che restituisce il risultato (stringa) o null
	timeout time.Duration
	res     chan scrapeRes
}

type scrapeRes struct {
	html string
	err  error
}

var (
	scrapeOnce sync.Once
	scrapeCh   = make(chan scrapeJob)
	pPeekMsg   = u32.NewProc("PeekMessageW")
)

func browserDump(u string) (string, error) {
	return browserRead(u, `document.documentElement.outerHTML.indexOf('__PRELOADED_STATE__')>=0 || !!document.querySelector('.builder__gear__name')`)
}

func browserRead(u, ready string) (string, error) {
	return browserRun(u, ready, "", "", 35*time.Second)
}

// browserRun apre la pagina, aspetta che "ready" sia vera, avvia "start" e attende che "poll" restituisca una stringa
func browserRun(u, ready, start, poll string, timeout time.Duration) (string, error) {
	scrapeOnce.Do(func() { go scrapeThread() })
	res := make(chan scrapeRes, 1)
	select {
	case scrapeCh <- scrapeJob{u, ready, start, poll, timeout, res}:
	case <-time.After(timeout + 3*time.Minute):
		return "", fmt.Errorf("lettore occupato")
	}
	r := <-res
	return r.html, r.err
}

func utf16At(p uintptr) string {
	if p == 0 {
		return ""
	}
	var buf []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(unsafe.Pointer(p + i))
		if c == 0 {
			break
		}
		buf = append(buf, c)
	}
	return syscall.UTF16ToString(buf)
}

func pump() {
	var m msgT
	for {
		r, _, _ := pPeekMsg.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1) // PM_REMOVE
		if r == 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func scrapeThread() {
	runtime.LockOSThread()
	fail := func(err error) {
		log.Printf("lettore pagine: %v", err)
		for j := range scrapeCh {
			j.res <- scrapeRes{"", err}
		}
	}
	pCoInitializeEx.Call(0, 2)
	dll := findWebViewDLL()
	if dll == "" {
		fail(fmt.Errorf("WebView2 non trovato"))
		return
	}
	dp, _ := syscall.UTF16PtrFromString(dll)
	mod, _, _ := pLoadLibraryEx.Call(uintptr(unsafe.Pointer(dp)), 0, 0x8)
	name := append([]byte("CreateWebViewEnvironmentWithOptionsInternal"), 0)
	create, _, _ := pGetProcAddress.Call(mod, uintptr(unsafe.Pointer(&name[0])))
	if mod == 0 || create == 0 {
		fail(fmt.Errorf("WebView2 non caricato"))
		return
	}
	hinst, _, _ := pGetModuleHandle.Call(0)
	cls, _ := syscall.UTF16PtrFromString("HellriseReader")
	wc := wndClassEx{lpfnWndProc: syscall.NewCallback(func(h, msg, wp, lp uintptr) uintptr {
		r, _, _ := pDefWindowProc.Call(h, msg, wp, lp)
		return r
	}), hInstance: hinst, lpszClassName: cls}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := pCreateWindowEx.Call(0x00000080|0x08000000, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(cls)), 0x80000000,
		uintptr(^uint32(3999)), uintptr(^uint32(3999)), 1366, 900, 0, 0, hinst, 0) // fuori dallo schermo, senza icona nella barra
	if hwnd == 0 {
		fail(fmt.Errorf("finestra del lettore non creata"))
		return
	}
	pShowWindow.Call(hwnd, 4) // SW_SHOWNOACTIVATE: visibile (fuori schermo) così la pagina non viene rallentata
	var wv, ctrl uintptr
	ready := make(chan bool, 1)
	ctrlH := newHandler(func(this, hr, c uintptr) uintptr {
		if hr != 0 || c == 0 {
			ready <- false
			return 0
		}
		comCall(c, 1)
		ctrl = c
		r := rect{0, 0, 1366, 900}
		comCall(c, 6, uintptr(unsafe.Pointer(&r)))
		comCall(c, 25, uintptr(unsafe.Pointer(&wv)))
		ready <- wv != 0
		return 0
	})
	envH := newHandler(func(this, hr, env uintptr) uintptr {
		if hr != 0 || env == 0 {
			ready <- false
			return 0
		}
		comCall(env, 3, hwnd, uintptr(unsafe.Pointer(ctrlH)))
		return 0
	})
	ud, _ := syscall.UTF16PtrFromString(filepath.Join(dataDir, "lettore"))
	if r, _, _ := syscall.SyscallN(create, 1, 0, uintptr(unsafe.Pointer(ud)), 0, uintptr(unsafe.Pointer(envH))); int32(r) < 0 {
		fail(fmt.Errorf("ambiente WebView2: 0x%08x", uint32(r)))
		return
	}
	ok := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		pump()
		select {
		case ok = <-ready:
			deadline = time.Now()
		default:
			time.Sleep(15 * time.Millisecond)
		}
	}
	if !ok || wv == 0 {
		fail(fmt.Errorf("WebView2 del lettore non avviato"))
		return
	}
	_ = ctrl
	log.Printf("lettore pagine pronto")
	var result string
	var got bool
	execH := newHandler(func(this, hr, str uintptr) uintptr {
		result, got = utf16At(str), true
		return 0
	})
	run := func(js string) (string, bool) {
		sp, _ := syscall.UTF16PtrFromString(js)
		got = false
		comCall(wv, 29, uintptr(unsafe.Pointer(sp)), uintptr(unsafe.Pointer(execH))) // ExecuteScript
		w := time.Now().Add(5 * time.Second)
		for !got && time.Now().Before(w) {
			pump()
			time.Sleep(10 * time.Millisecond)
		}
		if !got || result == "" || result == "null" {
			return "", false
		}
		var s string
		if json.Unmarshal([]byte(result), &s) != nil {
			return "", false
		}
		return s, true
	}
	wait := func(d time.Duration) {
		end := time.Now().Add(d)
		for time.Now().Before(end) {
			pump()
			time.Sleep(10 * time.Millisecond)
		}
	}
	for job := range scrapeCh {
		u, _ := syscall.UTF16PtrFromString(job.url)
		comCall(wv, 5, uintptr(unsafe.Pointer(u))) // Navigate
		wait(1200 * time.Millisecond)
		readyJS := "(function(){try{return (" + job.ready + ")?'1':null}catch(e){return null}})()"
		if job.start == "" {
			readyJS = "(function(){try{if(!(" + job.ready + "))return null;return document.documentElement.outerHTML;}catch(e){return null}})()"
		}
		var out string
		end := time.Now().Add(35 * time.Second)
		for time.Now().Before(end) && out == "" {
			if s, ok := run(readyJS); ok && s != "" {
				out = s
				break
			}
			wait(700 * time.Millisecond)
		}
		if out == "" {
			job.res <- scrapeRes{"", fmt.Errorf("pagina non pronta entro 35 secondi")}
			continue
		}
		if job.start != "" {
			run("(function(){try{" + job.start + "}catch(e){}return null})()")
			out = ""
			end = time.Now().Add(job.timeout)
			for time.Now().Before(end) && out == "" {
				wait(1000 * time.Millisecond)
				if s, ok := run("(function(){try{return (" + job.poll + ")}catch(e){return null}})()"); ok && s != "" {
					out = s
				}
			}
			if out == "" {
				job.res <- scrapeRes{"", fmt.Errorf("lettura non finita entro %v", job.timeout)}
				continue
			}
		}
		job.res <- scrapeRes{out, nil}
	}
}
