//go:build windows

package main

// Browser dei siti di build (Maxroll, D4Builds, Mobalytics) dentro la finestra di Hellrise:
// un secondo WebView2 sopra la pagina, nella zona che l'interfaccia gli indica. Usa il profilo
// di Hellrise, quindi non tocca i browser installati.

import (
	"log"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type brReq struct {
	URL  string
	R    rect
	Out  BrowserState
	Done bool
}

type BrowserState struct {
	Ready   bool   `json:"ready"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	CanBack bool   `json:"canBack"`
	CanFwd  bool   `json:"canFwd"`
	Visible bool   `json:"visible"`
}

var br struct {
	sync.Mutex
	ctrl, wv uintptr
	creating bool
	visible  bool
	bounds   rect
	pending  string
	keep     []any
	before   map[uintptr]bool // finestre figlie prima di creare il browser
	child    uintptr          // la finestra del browser (va tenuta sopra la pagina dell'app)
}

var pGetWindow = u32.NewProc("GetWindow")

func childWindows(h uintptr) map[uintptr]bool {
	out := map[uintptr]bool{}
	c, _, _ := pGetWindow.Call(h, 5) // GW_CHILD
	for c != 0 {
		out[c] = true
		c, _, _ = pGetWindow.Call(c, 2) // GW_HWNDNEXT
	}
	return out
}

// il browser dei siti sta sopra la pagina dell'app
func brRaise() {
	if br.child == 0 && br.before != nil {
		for c := range childWindows(wvHwnd) {
			if !br.before[c] {
				br.child = c
			}
		}
	}
	if br.child != 0 {
		pSetWindowPos.Call(br.child, 0, 0, 0, 0, 0, 0x0001|0x0002|0x0010) // HWND_TOP, NOSIZE|NOMOVE|NOACTIVATE
	}
}

// chiamata dall'interfaccia (thread HTTP): esegue l'operazione sul thread della finestra
func browserOp(op int, u string, r rect) (BrowserState, bool) {
	wvMu.Lock()
	h := wvHwnd
	wvMu.Unlock()
	if h == 0 || wvEnv == 0 {
		return BrowserState{}, false
	}
	req := &brReq{URL: u, R: r}
	pSendMessage.Call(h, wmApp+2, uintptr(op), uintptr(unsafe.Pointer(req)))
	return req.Out, req.Done
}

func coStr(p uintptr) string {
	if p == 0 {
		return ""
	}
	s := utf16At(p)
	pCoTaskMemFree.Call(p)
	return s
}

var pCoTaskMemFree = ole32.NewProc("CoTaskMemFree")

func brNavigate(wv uintptr, u string) {
	if wv == 0 || u == "" {
		return
	}
	p, _ := syscall.UTF16PtrFromString(u)
	comCall(wv, 5, uintptr(unsafe.Pointer(p)))
}

func brHandle(op int, lp uintptr) uintptr {
	req := (*brReq)(unsafe.Pointer(lp))
	req.Done = true
	switch op {
	case brShow, brNav:
		if op == brShow {
			br.bounds = req.R
			br.visible = true
		}
		if br.ctrl == 0 {
			if req.URL != "" {
				br.pending = req.URL
			}
			brCreate()
			break
		}
		if op == brShow {
			comCall(br.ctrl, 6, uintptr(unsafe.Pointer(&br.bounds))) // put_Bounds
			comCall(br.ctrl, 4, 1)                                   // put_IsVisible
		}
		if req.URL != "" {
			brNavigate(br.wv, req.URL)
		}
	case brHide:
		br.visible = false
		if br.ctrl != 0 {
			comCall(br.ctrl, 4, 0)
		}
	case brBack:
		if br.wv != 0 {
			comCall(br.wv, 40)
		}
	case brFwd:
		if br.wv != 0 {
			comCall(br.wv, 41)
		}
	case brReload:
		if br.wv != 0 {
			comCall(br.wv, 31)
		}
	}
	// stato attuale (indirizzo, titolo, avanti/indietro)
	st := BrowserState{Ready: br.wv != 0, Visible: br.visible}
	if br.wv != 0 {
		var p uintptr
		comCall(br.wv, 4, uintptr(unsafe.Pointer(&p))) // get_Source
		st.URL = coStr(p)
		p = 0
		comCall(br.wv, 48, uintptr(unsafe.Pointer(&p))) // get_DocumentTitle
		st.Title = coStr(p)
		var b int32
		comCall(br.wv, 38, uintptr(unsafe.Pointer(&b)))
		st.CanBack = b != 0
		b = 0
		comCall(br.wv, 39, uintptr(unsafe.Pointer(&b)))
		st.CanFwd = b != 0
	}
	req.Out = st
	return 0
}

// crea il secondo WebView2 (una volta sola)
func brCreate() {
	if br.creating || wvEnv == 0 || wvHwnd == 0 {
		return
	}
	br.creating = true
	br.before = childWindows(wvHwnd)
	var ctrlH *comHandler
	ctrlH = newHandler(func(this, hr, ctrl uintptr) uintptr {
		br.creating = false
		if hr != 0 || ctrl == 0 {
			return 0
		}
		comCall(ctrl, 1) // AddRef
		var wv uintptr
		comCall(ctrl, 25, uintptr(unsafe.Pointer(&wv))) // get_CoreWebView2
		if wv == 0 {
			return 0
		}
		br.ctrl, br.wv = ctrl, wv
		var st uintptr
		comCall(wv, 3, uintptr(unsafe.Pointer(&st))) // get_Settings
		if st != 0 {
			comCall(st, 10, 0) // niente barra di stato
		}
		// i link che aprirebbero una nuova finestra restano qui dentro
		nw := newHandler(func(this, sender, args uintptr) uintptr {
			if args == 0 {
				return 0
			}
			var p uintptr
			comCall(args, 3, uintptr(unsafe.Pointer(&p))) // get_Uri
			u := coStr(p)
			if strings.HasPrefix(u, "http") {
				comCall(args, 6, 1) // put_Handled
				brNavigate(br.wv, u)
			}
			return 0
		})
		br.keep = append(br.keep, nw)
		var tok int64
		comCall(wv, 44, uintptr(unsafe.Pointer(nw)), uintptr(unsafe.Pointer(&tok))) // add_NewWindowRequested
		comCall(ctrl, 6, uintptr(unsafe.Pointer(&br.bounds)))
		if br.visible {
			comCall(ctrl, 4, 1)
		} else {
			comCall(ctrl, 4, 0)
		}
		brRaise()
		log.Printf("browser dei siti pronto (finestra 0x%x)", br.child)
		u := br.pending
		if u == "" {
			u = "https://maxroll.gg/d4/build-guides"
		}
		brNavigate(wv, u)
		return 0
	})
	br.keep = append(br.keep, ctrlH)
	comCall(wvEnv, 3, wvHwnd, uintptr(unsafe.Pointer(ctrlH))) // CreateCoreWebView2Controller
}
