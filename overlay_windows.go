//go:build windows

package main

// Overlay in gioco disegnato con GDI su una finestra trasparente sopra Diablo IV (finestra senza bordi):
// timer degli eventi e guida paragon della build scelta. Non legge e non tocca il gioco.
// Si vede solo quando Diablo IV è in primo piano (o sempre, se scelto nelle impostazioni).
//
// Scorciatoie (attive solo con l'overlay visibile):
//   Ctrl+Shift+O  mostra/nasconde tutto      Ctrl+Shift+P  paragon      Ctrl+Shift+T  timer
//   Ctrl+Shift+E  sposta i riquadri          Ctrl+Shift+←/→  tavola precedente/successiva
//   Ctrl+Shift+↑/↓  fase del paragon (es. Rush → Endgame)

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	pSetLayeredAttr      = u32.NewProc("SetLayeredWindowAttributes")
	pInvalidateRect      = u32.NewProc("InvalidateRect")
	pBeginPaint          = u32.NewProc("BeginPaint")
	pEndPaint            = u32.NewProc("EndPaint")
	pSetTimer            = u32.NewProc("SetTimer")
	pRegisterHotKey      = u32.NewProc("RegisterHotKey")
	pUnregisterHotKey    = u32.NewProc("UnregisterHotKey")
	pFillRect            = u32.NewProc("FillRect")
	pDrawText            = u32.NewProc("DrawTextW")
	pGetForegroundWindow = u32.NewProc("GetForegroundWindow")
	pCreateCompatibleDC  = g32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBmp = g32.NewProc("CreateCompatibleBitmap")
	pSelectObject        = g32.NewProc("SelectObject")
	pDeleteObject        = g32.NewProc("DeleteObject")
	pDeleteDC            = g32.NewProc("DeleteDC")
	pBitBlt              = g32.NewProc("BitBlt")
	pCreatePen           = g32.NewProc("CreatePen")
	pRoundRect           = g32.NewProc("RoundRect")
	pCreateFont          = g32.NewProc("CreateFontW")
	pSetTextColor        = g32.NewProc("SetTextColor")
	pSetBkMode           = g32.NewProc("SetBkMode")
	pGetStockObject      = g32.NewProc("GetStockObject")
	pRectangle           = g32.NewProc("Rectangle")
)

const (
	wsExLayered     = 0x00080000
	wsExTransparent = 0x00000020
	wsExTopmost     = 0x00000008
	wsExToolWindow  = 0x00000080
	wsExNoActivate  = 0x08000000
	wsPopup         = 0x80000000
	lwaColorKey     = 0x1
	dtLeft          = 0x0
	dtCenter        = 0x1
	dtRight         = 0x2
	dtVCenter       = 0x4
	dtSingleLine    = 0x20
	dtNoPrefix      = 0x800
	dtEndEllipsis   = 0x8000
	srcCopy         = 0x00CC0020
	keyColor        = 0x00FF00FF // magenta = trasparente
	modControl      = 0x2
	modShift        = 0x4
	modNoRepeat     = 0x4000
)

type paintStruct struct {
	hdc      uintptr
	fErase   int32
	rcPaint  rect
	fRestore int32
	fIncUp   int32
	reserved [32]byte
}

var (
	ovHwnd    uintptr
	ovW, ovH  int32
	ovVisible bool
	ovHotkeys bool
	ovEdit    bool
	ovHideAll bool
	ovFonts   = map[int]uintptr{}
	ovHits    = map[string]rect{}
	ovDrag    string
	ovDragDX  int32
	ovDragDY  int32
)

// build del paragon in memoria (si ricarica quando cambia il planner scelto)
var ovPar struct {
	sync.Mutex
	id    string
	build *PBuild
	err   string
}

func overlayParagon() (*PBuild, string) {
	o := getSettings().Overlay
	ovPar.Lock()
	defer ovPar.Unlock()
	if o.Planner == "" {
		return nil, ""
	}
	if ovPar.id != o.Planner || (ovPar.build == nil && ovPar.err == "") {
		ovPar.id, ovPar.build, ovPar.err = o.Planner, nil, ""
		id := o.Planner
		go func() {
			b, err := plannerBuild(id)
			ovPar.Lock()
			if ovPar.id == id {
				ovPar.build = b
				if err != nil {
					ovPar.err = err.Error()
				}
			}
			ovPar.Unlock()
		}()
	}
	return ovPar.build, ovPar.err
}

func reloadOverlayParagon() {
	ovPar.Lock()
	ovPar.id, ovPar.build, ovPar.err = "", nil, ""
	ovPar.Unlock()
}

func col(hex uint32) uintptr { // 0xRRGGBB → COLORREF (evita il magenta trasparente)
	r, g, b := hex>>16&0xFF, hex>>8&0xFF, hex&0xFF
	if r == 255 && g == 0 && b == 255 {
		g = 1
	}
	return uintptr(r | g<<8 | b<<16)
}

func ovFont(size int, bold bool) uintptr {
	key := size * 2
	if bold {
		key++
	}
	if f, ok := ovFonts[key]; ok {
		return f
	}
	w := 500
	if bold {
		w = 700
	}
	face, _ := syscall.UTF16PtrFromString("Segoe UI")
	f, _, _ := pCreateFont.Call(uintptr(-size), 0, 0, 0, uintptr(w), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	ovFonts[key] = f
	return f
}

func ovText(hdc uintptr, s string, r rect, color uintptr, f uintptr, flags uintptr) {
	p, err := syscall.UTF16FromString(s)
	if err != nil || len(p) <= 1 {
		return
	}
	pSelectObject.Call(hdc, f)
	pSetTextColor.Call(hdc, color)
	pDrawText.Call(hdc, uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)-1), uintptr(unsafe.Pointer(&r)), flags|dtSingleLine|dtVCenter|dtNoPrefix|dtEndEllipsis)
}

func ovRound(hdc uintptr, r rect, rad int32, fill, border uintptr) {
	br, _, _ := pCreateSolidBrush.Call(fill)
	pen, _, _ := pCreatePen.Call(0, 1, border)
	ob, _, _ := pSelectObject.Call(hdc, br)
	op, _, _ := pSelectObject.Call(hdc, pen)
	pRoundRect.Call(hdc, uintptr(r.l), uintptr(r.t), uintptr(r.r), uintptr(r.b), uintptr(rad), uintptr(rad))
	pSelectObject.Call(hdc, ob)
	pSelectObject.Call(hdc, op)
	pDeleteObject.Call(br)
	pDeleteObject.Call(pen)
}

func ovFill(hdc uintptr, r rect, color uintptr) {
	br, _, _ := pCreateSolidBrush.Call(color)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), br)
	pDeleteObject.Call(br)
}

func ovDash(hdc uintptr, r rect, color uintptr) {
	pen, _, _ := pCreatePen.Call(1, 1, color)
	nb, _, _ := pGetStockObject.Call(5)
	op, _, _ := pSelectObject.Call(hdc, pen)
	ob, _, _ := pSelectObject.Call(hdc, nb)
	pRectangle.Call(hdc, uintptr(r.l-4), uintptr(r.t-4), uintptr(r.r+4), uintptr(r.b+4))
	pSelectObject.Call(hdc, op)
	pSelectObject.Call(hdc, ob)
	pDeleteObject.Call(pen)
}

func ovLeft(s int64) string {
	if s < 0 {
		s = 0
	}
	h, m, x := s/3600, s%3600/60, s%60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%d:%02d", m, x)
}

func pick(it, en string) string {
	if getSettings().Lang == "en" && en != "" {
		return en
	}
	if it != "" {
		return it
	}
	return en
}

// colori del tema (brace su vetro scuro, in tinta unita perché l'overlay non ha trasparenze)
var (
	cPanel  = col(0x17100E)
	cHead   = col(0x22160F)
	cBorder = col(0x4A2A1C)
	cInk    = col(0xF6EEE8)
	cMute   = col(0xA8968C)
	cAcc    = col(0xFF7A3A)
	cAcc2   = col(0xFFB14A)
)

func paintOverlay(hdc uintptr) {
	mem, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBmp.Call(hdc, uintptr(ovW), uintptr(ovH))
	old, _, _ := pSelectObject.Call(mem, bmp)
	ovFill(mem, rect{0, 0, ovW, ovH}, keyColor)
	pSetBkMode.Call(mem, 1)
	o := getSettings().Overlay
	k := float64(ovH) / 1080 * o.Scale
	if k <= 0 {
		k = float64(ovH) / 1080
	}
	sc := func(v float64) int32 { return int32(v*k + 0.5) }
	hits := map[string]rect{}
	if o.Timers.On {
		hits["timers"] = drawTimers(mem, o, sc, k)
	}
	if o.Paragon.On {
		hits["paragon"] = drawParagon(mem, o, sc, k)
	}
	if ovEdit {
		for _, r := range hits {
			ovDash(mem, r, cAcc2)
		}
		ovRound(mem, rect{ovW/2 - sc(230), sc(14), ovW/2 + sc(230), sc(50)}, sc(14), cHead, cAcc)
		ovText(mem, "Trascina i riquadri · Ctrl+Shift+E per finire", rect{ovW/2 - sc(230), sc(14), ovW/2 + sc(230), sc(50)}, cInk, ovFont(int(13*k), true), dtCenter)
	}
	ovHits = hits
	pBitBlt.Call(hdc, 0, 0, uintptr(ovW), uintptr(ovH), mem, 0, 0, srcCopy)
	pSelectObject.Call(mem, old)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mem)
}

func widgetOrigin(w OWidget, width, height int32) (int32, int32) {
	x, y := int32(w.X*float64(ovW)), int32(w.Y*float64(ovH))
	if x+width > ovW {
		x = ovW - width
	}
	if y+height > ovH {
		y = ovH - height
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func drawTimers(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	evs := upcomingEvents(time.Now(), 6)
	type row struct {
		title, sub, right string
		live              bool
		color             uintptr
	}
	var rows []row
	now := time.Now().Unix()
	for _, kind := range []string{"boss", "helltide", "legion"} {
		if kind == "boss" && !o.TBoss || kind == "helltide" && !o.THelltide || kind == "legion" && !o.TLegion {
			continue
		}
		for _, e := range evs {
			if e.Kind != kind || e.End < now {
				continue
			}
			r := row{}
			switch kind {
			case "boss":
				r.title, r.color = e.Title, col(0xFF6A3D)
				var z []string
				for _, x := range e.Zones {
					z = append(z, x.Name)
				}
				r.sub = strings.Join(z, " · ")
				if e.Guess {
					r.sub = "orario stimato"
				}
			case "helltide":
				r.title, r.color = "Marea infernale", col(0xFF3D6A)
			case "legion":
				r.title, r.color = "Legione", col(0xB088FF)
			}
			if now >= e.Start {
				r.live, r.right = true, "fine "+ovLeft(e.End-now)
			} else {
				r.right = ovLeft(e.Start - now)
				if r.sub == "" {
					r.sub = "alle " + time.Unix(e.Start, 0).Format("15:04")
				}
			}
			rows = append(rows, r)
			break
		}
	}
	w := sc(270)
	rowH := sc(40)
	h := sc(30) + rowH*int32(len(rows)) + sc(8)
	if len(rows) == 0 {
		h = sc(30) + sc(30)
	}
	x, y := widgetOrigin(o.Timers, w, h)
	ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
	ovText(hdc, "HELLRISE · EVENTI", rect{x + sc(14), y + sc(4), x + w - sc(10), y + sc(28)}, cMute, ovFont(int(10.5*k), true), dtLeft)
	ry := y + sc(30)
	if len(rows) == 0 {
		ovText(hdc, "Nessun evento selezionato", rect{x + sc(14), ry, x + w, ry + sc(24)}, cMute, ovFont(int(12*k), false), dtLeft)
	}
	for _, r := range rows {
		ovRound(hdc, rect{x + sc(12), ry + sc(10), x + sc(22), ry + sc(20)}, sc(10), r.color, r.color)
		ovText(hdc, r.title, rect{x + sc(30), ry + sc(2), x + w - sc(90), ry + sc(20)}, cInk, ovFont(int(13*k), true), dtLeft)
		if r.sub != "" {
			ovText(hdc, r.sub, rect{x + sc(30), ry + sc(19), x + w - sc(80), ry + sc(36)}, cMute, ovFont(int(11*k), false), dtLeft)
		}
		c := cInk
		if r.live {
			c = cAcc2
		}
		ovText(hdc, r.right, rect{x + w - sc(110), ry + sc(4), x + w - sc(14), ry + sc(30)}, c, ovFont(int(15*k), true), dtRight)
		ry += rowH
	}
	return rect{x, y, x + w, y + h}
}

// colori delle caselle paragon: [non presa, presa]
var cellCol = map[byte][2]uintptr{
	'1': {col(0x2A2422), col(0xD8CFC8)}, // normale
	'2': {col(0x1C2A44), col(0x4F8BFF)}, // magica
	'3': {col(0x3A321A), col(0xFFD24A)}, // rara
	'4': {col(0x40200F), col(0xFF7A2A)}, // leggendaria
	'5': {col(0x333333), col(0xFFFFFF)}, // porta
	'6': {col(0x2C1A40), col(0xC07BFF)}, // glifo
	'7': {col(0x1A3A2A), col(0x7BE0A5)}, // inizio
}

func drawParagon(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	cell := o.Cell
	if cell <= 0 {
		cell = 13
	}
	cs := sc(float64(cell))
	if cs < 4 {
		cs = 4
	}
	b, perr := overlayParagon()
	var board *PBoard
	var step *PStep
	var variant *PVariant
	if b != nil && len(b.Variants) > 0 {
		vi := clampInt(o.Variant, 0, len(b.Variants)-1)
		variant = &b.Variants[vi]
		if len(variant.Paragon) > 0 {
			si := clampInt(o.Step, 0, len(variant.Paragon)-1)
			step = &variant.Paragon[si]
			if len(step.Boards) > 0 {
				board = &step.Boards[clampInt(o.Board, 0, len(step.Boards)-1)]
			}
		}
	}
	gw := int32(21) * cs
	if board != nil {
		gw = int32(board.Width) * cs
	}
	w := gw + sc(28)
	if w < sc(300) {
		w = sc(300)
	}
	head := sc(64)
	foot := sc(52)
	h := head + gw + foot
	if board == nil {
		h = head + sc(40)
	}
	x, y := widgetOrigin(o.Paragon, w, h)
	ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
	f10, f12, f13 := ovFont(int(10.5*k), true), ovFont(int(12*k), false), ovFont(int(14*k), true)
	title := "HELLRISE · PARAGON"
	if step != nil {
		vi := clampInt(o.Variant, 0, len(b.Variants)-1)
		si := clampInt(o.Step, 0, len(variant.Paragon)-1)
		title = fmt.Sprintf("PARAGON · %s · %s · TAVOLA %d/%d", strings.ToUpper(b.Variants[vi].Name), strings.ToUpper(variant.Paragon[si].Name), clampInt(o.Board, 0, len(step.Boards)-1)+1, len(step.Boards))
	}
	ovText(hdc, title, rect{x + sc(14), y + sc(4), x + w - sc(10), y + sc(26)}, cMute, f10, dtLeft)
	if board == nil {
		msg := "Scegli una build Maxroll in Hellrise → Overlay"
		switch {
		case perr != "":
			msg = "Planner non disponibile: " + perr
		case o.Planner != "" && b == nil:
			msg = "Carico il paragon…"
		case b != nil:
			msg = "Questa variante non ha un paragon"
		}
		ovText(hdc, msg, rect{x + sc(14), y + sc(30), x + w - sc(10), y + sc(60)}, cInk, f12, dtLeft)
		return rect{x, y, x + w, y + h}
	}
	name := pick(board.Name, board.En)
	ovText(hdc, name, rect{x + sc(14), y + sc(24), x + w - sc(90), y + sc(44)}, cInk, f13, dtLeft)
	rotTxt := fmt.Sprintf("ruota %d°", board.Rotation*90)
	if board.Rotation == 0 {
		rotTxt = "nessuna rotazione"
	}
	ovText(hdc, rotTxt, rect{x + w - sc(150), y + sc(24), x + w - sc(14), y + sc(44)}, cAcc2, ovFont(int(12*k), true), dtRight)
	sub := fmt.Sprintf("%d punti", board.Points)
	if g := pick(board.Glyph, board.GlyphEn); g != "" {
		sub = "Glifo: " + g
		if board.GlyphLvl > 0 {
			sub += fmt.Sprintf(" (liv. %d)", board.GlyphLvl)
		}
		sub += fmt.Sprintf(" · %d punti", board.Points)
	}
	ovText(hdc, sub, rect{x + sc(14), y + sc(42), x + w - sc(14), y + sc(60)}, cMute, f12, dtLeft)
	gx := x + (w-gw)/2
	gy := y + head
	bw := board.Width
	gap := int32(1)
	if cs < 8 {
		gap = 0
	}
	for i := 0; i < len(board.Cells); i++ {
		c := board.Cells[i]
		if c == '0' {
			continue
		}
		cc, ok := cellCol[c]
		if !ok {
			continue
		}
		cx, cy := gx+int32(i%bw)*cs, gy+int32(i/bw)*cs
		r := rect{cx + gap, cy + gap, cx + cs - gap, cy + cs - gap}
		if board.Sel[i] == '1' {
			ovFill(hdc, r, cc[1])
		} else {
			ovFill(hdc, r, cc[0])
		}
	}
	fy := gy + gw + sc(6)
	leg := ""
	for _, l := range board.Legend {
		p := strings.SplitN(l, "|", 2)
		en := ""
		if len(p) > 1 {
			en = p[1]
		}
		leg = "Leggendario: " + pick(p[0], en)
		break
	}
	if leg != "" {
		ovText(hdc, leg, rect{x + sc(14), fy, x + w - sc(14), fy + sc(20)}, cAcc, ovFont(int(12*k), true), dtLeft)
	}
	ovText(hdc, "Ctrl+Shift+←/→ tavola · ↑/↓ fase · P nascondi", rect{x + sc(14), fy + sc(22), x + w - sc(14), fy + sc(42)}, cMute, ovFont(int(10.5*k), false), dtLeft)
	return rect{x, y, x + w, y + h}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func diabloForeground() bool {
	h, _, _ := pGetForegroundWindow.Call()
	if h == 0 {
		return false
	}
	t := make([]uint16, 128)
	pGetWindowText.Call(h, uintptr(unsafe.Pointer(&t[0])), 128)
	title := syscall.UTF16ToString(t)
	return title == "Diablo IV" || strings.HasPrefix(title, "Diablo IV ")
}

func setOverlayHotkeys(on bool) {
	if on == ovHotkeys {
		return
	}
	ovHotkeys = on
	keys := map[int]uintptr{1: 'O', 2: 'P', 3: 'T', 4: 'E', 5: 0x27, 6: 0x25, 7: 0x26, 8: 0x28}
	for id, vk := range keys {
		if on {
			pRegisterHotKey.Call(ovHwnd, uintptr(id), modControl|modShift|modNoRepeat, vk)
		} else {
			pUnregisterHotKey.Call(ovHwnd, uintptr(id))
		}
	}
}

func setOverlayEdit(on bool) {
	ovEdit = on
	ex, _, _ := pGetWindowLongPtr.Call(ovHwnd, ^uintptr(19)) // GWL_EXSTYLE (-20)
	if on {
		ex &^= wsExTransparent
	} else {
		ex |= wsExTransparent
	}
	pSetWindowLongPtr.Call(ovHwnd, ^uintptr(19), ex)
}

func overlayTick() {
	o := getSettings().Overlay
	show := o.Enabled && !ovHideAll && (o.Always || diabloForeground() || ovEdit)
	hot := o.Enabled && (o.Always || diabloForeground() || ovEdit)
	setOverlayHotkeys(hot)
	if show != ovVisible {
		ovVisible = show
		if show {
			pShowWindow.Call(ovHwnd, 8) // SW_SHOWNA
		} else {
			pShowWindow.Call(ovHwnd, 0)
			if ovEdit {
				setOverlayEdit(false)
			}
		}
	}
	if show {
		pInvalidateRect.Call(ovHwnd, 0, 0)
	}
}

func overlayHotkey(id int) {
	switch id {
	case 1:
		ovHideAll = !ovHideAll
	case 2:
		updateSettings(func(s *Settings) { s.Overlay.Paragon.On = !s.Overlay.Paragon.On })
	case 3:
		updateSettings(func(s *Settings) { s.Overlay.Timers.On = !s.Overlay.Timers.On })
	case 4:
		setOverlayEdit(!ovEdit)
	case 5, 6, 7, 8:
		b, _ := overlayParagon()
		updateSettings(func(s *Settings) {
			o := &s.Overlay
			nb, ns := 1, 1
			if b != nil && len(b.Variants) > 0 {
				v := b.Variants[clampInt(o.Variant, 0, len(b.Variants)-1)]
				ns = len(v.Paragon)
				if ns > 0 {
					nb = len(v.Paragon[clampInt(o.Step, 0, ns-1)].Boards)
				}
			}
			switch id {
			case 5:
				o.Board = clampInt(o.Board+1, 0, max(nb-1, 0))
			case 6:
				o.Board = clampInt(o.Board-1, 0, max(nb-1, 0))
			case 7:
				o.Step = clampInt(o.Step-1, 0, max(ns-1, 0))
			case 8:
				o.Step = clampInt(o.Step+1, 0, max(ns-1, 0))
			}
		})
	}
	overlayTick()
}

func ovWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case 0x000F: // WM_PAINT
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		paintOverlay(hdc)
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case 0x0014: // WM_ERASEBKGND
		return 1
	case 0x0113: // WM_TIMER
		overlayTick()
		return 0
	case 0x0312: // WM_HOTKEY
		overlayHotkey(int(wparam))
		return 0
	case 0x0201: // WM_LBUTTONDOWN: inizia a trascinare un riquadro
		if ovEdit {
			x, y := int32(int16(lparam&0xFFFF)), int32(int16(lparam>>16&0xFFFF))
			for key, r := range ovHits {
				if x >= r.l && x <= r.r && y >= r.t && y <= r.b {
					ovDrag, ovDragDX, ovDragDY = key, x-r.l, y-r.t
					pSetCapture.Call(hwnd)
				}
			}
		}
		return 0
	case 0x0200: // WM_MOUSEMOVE
		if ovDrag != "" {
			x, y := int32(int16(lparam&0xFFFF)), int32(int16(lparam>>16&0xFFFF))
			fx, fy := float64(x-ovDragDX)/float64(ovW), float64(y-ovDragDY)/float64(ovH)
			key := ovDrag
			setMu.Lock()
			if key == "timers" {
				settings.Overlay.Timers.X, settings.Overlay.Timers.Y = fx, fy
			} else {
				settings.Overlay.Paragon.X, settings.Overlay.Paragon.Y = fx, fy
			}
			setMu.Unlock()
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case 0x0202: // WM_LBUTTONUP
		if ovDrag != "" {
			ovDrag = ""
			pReleaseCapture.Call()
			saveSettings()
		}
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wparam, lparam)
	return r
}

func startOverlay() {
	runtime.LockOSThread()
	hinst, _, _ := pGetModuleHandle.Call(0)
	cls, _ := syscall.UTF16PtrFromString("HellriseOverlay")
	title, _ := syscall.UTF16PtrFromString("Hellrise Overlay")
	wc := wndClassEx{lpfnWndProc: syscall.NewCallback(ovWndProc), hInstance: hinst, lpszClassName: cls}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	ovW, ovH = screenSize()
	ovHwnd, _, _ = pCreateWindowEx.Call(wsExLayered|wsExTransparent|wsExTopmost|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), wsPopup, 0, 0, uintptr(ovW), uintptr(ovH), 0, 0, hinst, 0)
	if ovHwnd == 0 {
		log.Printf("overlay: finestra non creata")
		return
	}
	pSetLayeredAttr.Call(ovHwnd, keyColor, 0, lwaColorKey)
	pSetTimer.Call(ovHwnd, 1, 500, 0)
	var m msgT
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
