//go:build windows

package main

// Finestra dell'app tutta nostra (senza barra di Windows, angoli arrotondati) con dentro
// il motore WebView2 di Edge, già presente su Windows 10/11. Se non è disponibile si
// torna alla finestra "app" di Edge.

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	ole32              = syscall.NewLazyDLL("ole32.dll")
	pCoInitializeEx    = ole32.NewProc("CoInitializeEx")
	pGetClientRect     = u32.NewProc("GetClientRect")
	pDestroyWindow     = u32.NewProc("DestroyWindow")
	pPostQuitMessage   = u32.NewProc("PostQuitMessage")
	pLoadIcon          = u32.NewProc("LoadIconW")
	pMonitorFromWindow = u32.NewProc("MonitorFromWindow")
	pGetMonitorInfo    = u32.NewProc("GetMonitorInfoW")
	pIsIconic          = u32.NewProc("IsIconic")
	pLoadLibraryEx     = k32.NewProc("LoadLibraryExW")
	pGetProcAddress    = k32.NewProc("GetProcAddress")

	wvMu         sync.Mutex
	wvHwnd       uintptr
	wvController uintptr
	wvEnv        uintptr
	wvURL        string
	wvFailed     atomic.Bool
	wvMaxed      bool
	wvRestore    rect
	wvKeep       []any
	ncCount      atomic.Int64 // oggetti COM passati a WebView2: il GC non deve liberarli
)

const wmApp = 0x8000

// ---------- piccoli oggetti COM fatti a mano ----------

type comHandler struct {
	vtbl *[4]uintptr
}

var (
	cbQI      = syscall.NewCallback(func(this, riid, ppv uintptr) uintptr { *(*uintptr)(unsafe.Pointer(ppv)) = this; return 0 })
	cbAddRef  = syscall.NewCallback(func(this uintptr) uintptr { return 1 })
	cbRelease = syscall.NewCallback(func(this uintptr) uintptr { return 1 })
)

func newHandler(invoke func(this, hr, obj uintptr) uintptr) *comHandler {
	v := &[4]uintptr{cbQI, cbAddRef, cbRelease, syscall.NewCallback(invoke)}
	h := &comHandler{vtbl: v}
	wvKeep = append(wvKeep, h, v)
	return h
}

func comCall(obj uintptr, idx int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

// ---------- runtime WebView2 ----------

func findWebViewDLL() string {
	var roots []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if b := os.Getenv(env); b != "" {
			roots = append(roots, filepath.Join(b, "Microsoft", "EdgeWebView", "Application"), filepath.Join(b, "Microsoft", "Edge", "Application"))
		}
	}
	for _, root := range roots {
		ents, _ := os.ReadDir(root)
		var vers []string
		for _, e := range ents {
			if e.IsDir() && len(e.Name()) > 0 && e.Name()[0] >= '0' && e.Name()[0] <= '9' {
				vers = append(vers, e.Name())
			}
		}
		sort.Slice(vers, func(i, j int) bool { return verLess(vers[j], vers[i]) })
		for _, v := range vers {
			p := filepath.Join(root, v, "EBWebView", "x64", "EmbeddedBrowserWebView.dll")
			if fileExists(p) {
				return p
			}
		}
	}
	return ""
}

func verLess(a, b string) bool {
	var x, y [4]int
	parse := func(s string, o *[4]int) {
		i := 0
		for _, c := range s {
			if c == '.' {
				i++
				if i > 3 {
					return
				}
			} else if c >= '0' && c <= '9' {
				o[i] = o[i]*10 + int(c-'0')
			}
		}
	}
	parse(a, &x)
	parse(b, &y)
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

// ---------- finestra ----------

func wvWndProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case 0x0083: // WM_NCCALCSIZE: tutta la finestra è area della pagina (niente barra)
		ncCount.Add(1)
		if wp != 0 {
			return 0
		}
	case 0x0005: // WM_SIZE
		wvResize()
		return 0
	case 0x0024: // WM_GETMINMAXINFO
		mm := (*[10]int32)(unsafe.Pointer(lp))
		mm[6], mm[7] = 900, 560 // ptMinTrackSize
		return 0
	case 0x0010: // WM_CLOSE
		pDestroyWindow.Call(h)
		return 0
	case 0x0002: // WM_DESTROY
		log.Printf("finestra WebView2 chiusa")
		wvMu.Lock()
		if wvController != 0 {
			comCall(wvController, 24) // Close
			wvController = 0
		}
		wvHwnd = 0
		wvMu.Unlock()
		pPostQuitMessage.Call(0)
		return 0
	case wmApp + 1: // azioni chieste dall'interfaccia
		wvAction(int(wp))
		return 0
	case wmApp + 2: // browser dei siti dentro l'app
		return brHandle(int(wp), lp)
	}
	r, _, _ := pDefWindowProc.Call(h, msg, wp, lp)
	return r
}

func wvResize() {
	wvMu.Lock()
	h, c := wvHwnd, wvController
	wvMu.Unlock()
	if h == 0 || c == 0 {
		return
	}
	var r rect
	pGetClientRect.Call(h, uintptr(unsafe.Pointer(&r)))
	comCall(c, 6, uintptr(unsafe.Pointer(&r))) // put_Bounds (RECT passato per indirizzo su x64)
}

func workArea(h uintptr) rect {
	mon, _, _ := pMonitorFromWindow.Call(h, 2)
	var mi struct {
		size      uint32
		mon, work rect
		flags     uint32
	}
	mi.size = uint32(unsafe.Sizeof(mi))
	pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi)))
	return mi.work
}

func wvAction(a int) {
	h := wvHwnd
	if h == 0 {
		return
	}
	switch a {
	case 1: // riduci
		pShowWindow.Call(h, 6)
	case 2: // ingrandisci / ripristina (sull'area di lavoro, senza coprire la barra delle applicazioni)
		if wvMaxed {
			r := wvRestore
			pSetWindowPos.Call(h, 0, uintptr(r.l), uintptr(r.t), uintptr(r.r-r.l), uintptr(r.b-r.t), 0x0004)
			wvMaxed = false
			setCorners(h, true)
		} else {
			pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&wvRestore)))
			w := workArea(h)
			pSetWindowPos.Call(h, 0, uintptr(w.l), uintptr(w.t), uintptr(w.r-w.l), uintptr(w.b-w.t), 0x0004)
			wvMaxed = true
			setCorners(h, false)
		}
	case 3: // chiudi
		pPostMessage.Call(h, 0x0010, 0, 0)
	}
}

func setCorners(h uintptr, round bool) {
	pref := uint32(2) // arrotondati
	if !round {
		pref = 1 // squadrati quando la finestra è ingrandita
	}
	pDwmSetWindowAttrib.Call(h, 33, uintptr(unsafe.Pointer(&pref)), 4)
}

// apre (o porta davanti) la finestra con WebView2; false se WebView2 non c'è
func openWebViewWindow(url string) bool {
	if wvFailed.Load() {
		return false
	}
	wvMu.Lock()
	if wvHwnd != 0 {
		h := wvHwnd
		wvMu.Unlock()
		if ic, _, _ := pIsIconic.Call(h); ic != 0 {
			pShowWindow.Call(h, 9)
		}
		pSetForeground.Call(h)
		return true
	}
	wvMu.Unlock()
	dll := findWebViewDLL()
	if dll == "" {
		wvFailed.Store(true)
		return false
	}
	res := make(chan bool, 1)
	go wvThread(dll, url, res)
	select {
	case ok := <-res:
		return ok
	case <-time.After(20 * time.Second):
		return wvHwnd != 0
	}
}

func wvThread(dll, url string, res chan bool) {
	runtime.LockOSThread()
	pCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
	dp, _ := syscall.UTF16PtrFromString(dll)
	mod, _, _ := pLoadLibraryEx.Call(uintptr(unsafe.Pointer(dp)), 0, 0x8)
	if mod == 0 {
		wvFailed.Store(true)
		res <- false
		return
	}
	name := append([]byte("CreateWebViewEnvironmentWithOptionsInternal"), 0)
	create, _, _ := pGetProcAddress.Call(mod, uintptr(unsafe.Pointer(&name[0])))
	if create == 0 {
		wvFailed.Store(true)
		res <- false
		return
	}

	hinst, _, _ := pGetModuleHandle.Call(0)
	cls, _ := syscall.UTF16PtrFromString("HellriseApp")
	title, _ := syscall.UTF16PtrFromString("Hellrise")
	icon, _, _ := pLoadIcon.Call(hinst, 1)
	cur, _, _ := pLoadCursor.Call(0, 32512)
	bg, _, _ := pCreateSolidBrush.Call(0x0C0607) // stesso colore di sfondo dell'app
	wc := wndClassEx{lpfnWndProc: syscall.NewCallback(wvWndProc), hInstance: hinst, lpszClassName: cls, hIcon: icon, hIconSm: icon, hCursor: cur, hbrBackground: bg}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	sw0, sh0 := screenSize()
	sw, sh := int(sw0), int(sh0)
	w, h := 1480, 920
	if w > sw-40 {
		w = sw - 40
	}
	if h > sh-60 {
		h = sh - 60
	}
	const style = 0x80000000 | 0x00040000 | 0x00020000 | 0x00010000 | 0x00080000 // POPUP|THICKFRAME|MINIMIZE|MAXIMIZE|SYSMENU: senza WS_CAPTION Windows non disegna la barra; THICKFRAME tiene l'ombra
	hwnd, _, _ := pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), style,
		uintptr((sw-w)/2), uintptr((sh-h)/2), uintptr(w), uintptr(h), 0, 0, hinst, 0)
	if hwnd == 0 {
		wvFailed.Store(true)
		res <- false
		return
	}
	dark := uint32(1)
	pDwmSetWindowAttrib.Call(hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	setCorners(hwnd, true)
	pSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpFrameChg) // ricalcola l'area: niente barra
	wvMu.Lock()
	wvHwnd, wvURL = hwnd, url
	wvMu.Unlock()

	ok := make(chan bool, 1)
	ctrlHandler := newHandler(func(this, hr, ctrl uintptr) uintptr {
		log.Printf("WebView2: controller 0x%08x", uint32(hr))
		if hr != 0 || ctrl == 0 {
			ok <- false
			return 0
		}
		comCall(ctrl, 1) // AddRef
		wvMu.Lock()
		wvController = ctrl
		wvMu.Unlock()
		wvResize()
		var wv uintptr
		comCall(ctrl, 25, uintptr(unsafe.Pointer(&wv))) // get_CoreWebView2
		if wv == 0 {
			ok <- false
			return 0
		}
		var st uintptr
		comCall(wv, 3, uintptr(unsafe.Pointer(&st))) // get_Settings
		if st != 0 {
			comCall(st, 10, 0) // put_IsStatusBarEnabled(false)
			comCall(st, 14, 0) // put_AreDefaultContextMenusEnabled(false)
		}
		u, _ := syscall.UTF16PtrFromString(wvURL)
		comCall(wv, 5, uintptr(unsafe.Pointer(u))) // Navigate
		ok <- true
		return 0
	})
	envHandler := newHandler(func(this, hr, env uintptr) uintptr {
		log.Printf("WebView2: ambiente 0x%08x", uint32(hr))
		if hr != 0 || env == 0 {
			ok <- false
			return 0
		}
		comCall(env, 1) // AddRef: serve anche per il browser dei siti dentro l'app
		wvEnv = env
		comCall(env, 3, hwnd, uintptr(unsafe.Pointer(ctrlHandler))) // CreateCoreWebView2Controller
		return 0
	})
	ud, _ := syscall.UTF16PtrFromString(filepath.Join(dataDir, "webview"))
	started := false
	for _, rt := range []uintptr{0, 1} {
		r, _, _ := syscall.SyscallN(create, 1, rt, uintptr(unsafe.Pointer(ud)), 0, uintptr(unsafe.Pointer(envHandler)))
		log.Printf("WebView2: avvio runtime tipo %d → 0x%08x (%s)", rt, uint32(r), dll)
		if int32(r) >= 0 {
			started = true
			break
		}
	}
	if !started {
		pDestroyWindow.Call(hwnd)
		wvFailed.Store(true)
		res <- false
		return
	}
	pShowWindow.Call(hwnd, 5)
	reported := false
	var m msgT
	for {
		select {
		case good := <-ok:
			if !reported {
				reported = true
				if !good {
					wvFailed.Store(true)
					pDestroyWindow.Call(hwnd)
				}
				res <- good
			}
		default:
		}
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	if !reported {
		res <- false
	}
}

func wvPost(a int) bool {
	wvMu.Lock()
	h := wvHwnd
	wvMu.Unlock()
	if h == 0 {
		return false
	}
	pPostMessage.Call(h, wmApp+1, uintptr(a), 0)
	return true
}

func windowKind() string {
	switch {
	case wvHwnd != 0:
		return "webview"
	case wvFailed.Load():
		return "edge (webview non disponibile)"
	}
	return "edge"
}

func wvDebug() any {
	h := wvHwnd
	if h == 0 {
		return nil
	}
	var wr, cr rect
	pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&wr)))
	pGetClientRect.Call(h, uintptr(unsafe.Pointer(&cr)))
	st, _, _ := pGetWindowLongPtr.Call(h, gwlStyle)
	return map[string]any{"win": []int32{wr.l, wr.t, wr.r, wr.b}, "client": []int32{cr.l, cr.t, cr.r, cr.b}, "style": st, "ncc": ncCount.Load()}
}
