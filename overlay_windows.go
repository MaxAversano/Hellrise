//go:build windows

package main

// Overlay in gioco disegnato con GDI su una finestra trasparente sopra Diablo IV (finestra senza bordi):
// timer degli eventi e guida paragon della build scelta. Non legge e non tocca il gioco.
// Si vede solo quando Diablo IV è in primo piano (o sempre, se scelto nelle impostazioni).
//
// Scorciatoie (attive solo con l'overlay visibile):
//   Ctrl+Shift+O  mostra/nasconde tutto      Ctrl+Shift+P  paragon      Ctrl+Shift+T  timer
//   Ctrl+Shift+E  sposta i riquadri          Ctrl+Shift+←/→  tavola precedente/successiva
//   tenendo premuto il tasto scelto (Alt sinistro di base) i riquadri si spostano col mouse
//   e il lucchetto in alto a destra li blocca
//   Ctrl+Shift+↑/↓  fase del paragon (es. Rush → Endgame)
//   Ctrl+Shift+R / Q  ruota la tavola del riquadro in senso orario / antiorario
//   Ctrl+Shift+M  automatico ↔ riquadro sempre visibile
// In automatico l'overlay riconosce da solo la tavola aperta nella schermata del paragon
// (grandezza delle caselle, posizione e rotazione) e ci disegna sopra le caselle da prendere.

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
	pEllipse             = g32.NewProc("Ellipse")
	pGetDC               = u32.NewProc("GetDC")
	pReleaseDC           = u32.NewProc("ReleaseDC")
	pGetDIBits           = g32.NewProc("GetDIBits")
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
	pGetAsyncKeyState = u32.NewProc("GetAsyncKeyState")
	ovMove            bool // tasto per spostare tenuto premuto
	ovAlpha           int
	ovLocks           = map[string]rect{}
	ovEye             rect
	ovLabel           rect
	ovTags            []rect
)

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
			var b *PBuild
			var err error
			// senza i dati di gioco le tavole restano vuote: si aspetta che siano caricati
			for i := 0; i < 120 && getGD() == nil; i++ {
				time.Sleep(time.Second)
			}
			if strings.HasPrefix(id, "d4b:") {
				b, err = d4bParagonBuild(strings.TrimPrefix(id, "d4b:"))
			} else {
				b, err = plannerBuild(id)
			}
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

var symFonts = map[int]uintptr{}

func symFont(size int) uintptr {
	if f, ok := symFonts[size]; ok {
		return f
	}
	face, _ := syscall.UTF16PtrFromString("Segoe UI Emoji")
	f, _, _ := pCreateFont.Call(uintptr(-size), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	symFonts[size] = f
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
	ovEye, ovLabel, ovTags = rect{}, rect{}, ovTags[:0]
	if o.Paragon.On {
		r := drawParagon(mem, o, sc, k)
		hits["paragon"] = r
		if r.r > r.l { // occhietto: apre e chiude il paragon (clic tenendo premuto il tasto per spostare, o Alt+Shift+P)
			e := rect{r.l + sc(6), r.t + sc(4), r.l + sc(30), r.t + sc(26)}
			if !o.ParMin {
				e = rect{r.r - sc(62), r.t + sc(4), r.r - sc(36), r.t + sc(26)}
			}
			ovRound(mem, e, sc(8), cHead, cBorder)
			sym := "\U0001F441"
			c := cAcc2
			if o.ParMin {
				c = cMute
			}
			ovText(mem, sym, e, c, symFont(int(12*k)), dtCenter)
			ovEye = e
		}
	}
	if o.Rotation.On && len(o.RotLines) > 0 {
		hits["rotation"] = drawRotation(mem, o, sc, k)
	}
	locks := map[string]rect{}
	if ovEdit || ovMove {
		for key, r := range hits {
			if r.r <= r.l {
				continue
			}
			locked := widgetOf(&o, key).Lock
			c := cAcc2
			if locked {
				c = cMute
			}
			ovDash(mem, r, c)
			lr := rect{r.r - sc(30), r.t + sc(4), r.r - sc(6), r.t + sc(26)}
			bg := cHead
			if locked {
				bg = cAcc
			}
			ovRound(mem, lr, sc(8), bg, cBorder)
			sym := "\U0001F513" // lucchetto aperto
			if locked {
				sym = "\U0001F512"
			}
			ovText(mem, sym, lr, cInk, symFont(int(12*k)), dtCenter)
			locks[key] = lr
		}
		msg := "Trascina i riquadri · lucchetto = blocca · rotellina = grandezza paragon"
		if ovEdit {
			msg += " · Ctrl+Shift+E per finire"
		}
		ovRound(mem, rect{ovW/2 - sc(260), sc(14), ovW/2 + sc(260), sc(50)}, sc(14), cHead, cAcc)
		ovText(mem, msg, rect{ovW/2 - sc(260), sc(14), ovW/2 + sc(260), sc(50)}, cInk, ovFont(int(13*k), true), dtCenter)
	}
	ovLocks = locks
	ovHits = hits
	ovDrawnMu.Lock()
	ovDrawn = ovDrawn[:0]
	for _, r := range hits {
		if r.r > r.l {
			ovDrawn = append(ovDrawn, r)
		}
	}
	ovDrawn = append(ovDrawn, ovLabel)
	ovDrawn = append(ovDrawn, ovTags...)
	ovDrawnMu.Unlock()
	pBitBlt.Call(hdc, 0, 0, uintptr(ovW), uintptr(ovH), mem, 0, 0, srcCopy)
	pSelectObject.Call(mem, old)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mem)
}

func widgetOf(o *OverlaySet, key string) *OWidget {
	switch key {
	case "timers":
		return &o.Timers
	case "rotation":
		return &o.Rotation
	}
	return &o.Paragon
}

// riquadro con la rotazione delle abilità della build scelta
func drawRotation(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	lines := o.RotLines
	if len(lines) > 12 {
		lines = lines[:12]
	}
	w := sc(330)
	rowH := sc(22)
	h := sc(32) + rowH*int32(len(lines)) + sc(10)
	x, y := widgetOrigin(o.Rotation, w, h)
	ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
	title := "ROTAZIONE"
	if o.RotTitle != "" {
		title += " · " + strings.ToUpper(o.RotTitle)
	}
	ovText(hdc, title, rect{x + sc(14), y + sc(5), x + w - sc(34), y + sc(28)}, cMute, ovFont(int(10.5*k), true), dtLeft)
	ry := y + sc(30)
	for i, l := range lines {
		ovText(hdc, fmt.Sprintf("%d", i+1), rect{x + sc(12), ry, x + sc(30), ry + rowH}, cAcc2, ovFont(int(12*k), true), dtLeft)
		ovText(hdc, l, rect{x + sc(32), ry, x + w - sc(10), ry + rowH}, cInk, ovFont(int(12.5*k), false), dtLeft|0x8000)
		ry += rowH
	}
	return rect{x, y, x + w, y + h}
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

type curBoard struct {
	b       *PBuild
	variant *PVariant
	step    *PStep
	board   *PBoard
	key     string // per ricordare la rotazione scelta
	rot     int    // rotazione effettiva (quarti di giro, orario)
	cells   string
	sel     string
}

func currentBoard(o OverlaySet) (curBoard, string) {
	var c curBoard
	b, perr := overlayParagon()
	c.b = b
	if b == nil || len(b.Variants) == 0 {
		return c, perr
	}
	vi := clampInt(o.Variant, 0, len(b.Variants)-1)
	c.variant = &b.Variants[vi]
	if len(c.variant.Paragon) == 0 {
		return c, perr
	}
	si := clampInt(o.Step, 0, len(c.variant.Paragon)-1)
	c.step = &c.variant.Paragon[si]
	if len(c.step.Boards) == 0 {
		return c, perr
	}
	bi := clampInt(o.Board, 0, len(c.step.Boards)-1)
	c.board = &c.step.Boards[bi]
	c.key = fmt.Sprintf("%s|%d|%d|%d", o.Planner, vi, si, bi)
	c.rot = c.board.Rotation
	if r, ok := o.Rot[c.key]; ok {
		c.rot = ((r % 4) + 4) % 4
	}
	c.cells = rotateGrid(c.board.RawCells, c.board.Width, c.rot)
	c.sel = rotateGrid(c.board.RawSel, c.board.Width, c.rot)
	if c.cells == "" {
		c.cells, c.sel = c.board.Cells, c.board.Sel
	}
	return c, perr
}

func drawParagon(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	if o.ParMin { // chiuso: resta solo l'occhietto
		w, h := sc(176), sc(30)
		x, y := widgetOrigin(o.Paragon, w, h)
		ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
		ovText(hdc, "Paragon · Alt+Shift+P", rect{x + sc(34), y, x + w - sc(8), y + h}, cMute, ovFont(int(11*k), true), dtLeft)
		return rect{x, y, x + w, y + h}
	}
	// automatico: se il paragon del gioco è aperto e riconosciuto, cornici sulla tavola vera;
	// altrimenti la guida con tutto il paragon della build fissata
	if o.Mode != "panel" {
		if d, ok := lastDetect(); ok && d.OK && d.Board >= 0 {
			return drawParagonAligned(hdc, o, sc, k)
		}
	}
	return drawParagonGuide(hdc, o, sc, k)
}

// "1 scatto (90° in senso orario)"
func rotPhrase(r int) string {
	switch ((r % 4) + 4) % 4 {
	case 1:
		return "ruotala di 1 scatto (90°, senso orario)"
	case 2:
		return "ruotala di 2 scatti (180°)"
	case 3:
		return "ruotala di 3 scatti (90° antiorario)"
	}
	return "nessuna rotazione"
}

// disegna una tavola (già ruotata) in un riquadro
func drawBoardCells(hdc uintptr, cells, sel string, w int, x, y int32, cs float64, gap int32) {
	for i := 0; i < len(cells) && i < len(sel); i++ {
		c := cells[i]
		cc, ok := cellCol[c]
		if !ok {
			continue
		}
		x0 := x + int32(float64(i%w)*cs)
		y0 := y + int32(float64(i/w)*cs)
		x1 := x + int32(float64(i%w+1)*cs)
		y1 := y + int32(float64(i/w+1)*cs)
		if x1-x0 > 3 {
			x0, y0, x1, y1 = x0+gap, y0+gap, x1-gap, y1-gap
		}
		if x1 <= x0 {
			x1 = x0 + 1
		}
		if y1 <= y0 {
			y1 = y0 + 1
		}
		if sel[i] == '1' {
			ovFill(hdc, rect{x0, y0, x1, y1}, cc[1])
		} else {
			ovFill(hdc, rect{x0, y0, x1, y1}, cc[0])
		}
	}
}

// guida: tutto il paragon (mappa delle tavole) + la tavola da fare adesso, con le istruzioni
func drawParagonGuide(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	cb, perr := currentBoard(o)
	b, board, step := cb.b, cb.board, cb.step
	f10, f12, f13 := ovFont(int(10.5*k), true), ovFont(int(12*k), false), ovFont(int(14*k), true)
	cell := float64(o.Cell)
	if cell <= 0 {
		cell = 13
	}
	side := sc(cell * 0.75 * 21) // lato della tavola grande e della mappa
	if side < sc(150) {
		side = sc(150)
	}
	pad := sc(14)
	w := side*2 + pad*3
	if board == nil {
		w = sc(360)
		h := sc(70)
		x, y := widgetOrigin(o.Paragon, w, h)
		ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
		ovText(hdc, "HELLRISE · PARAGON", rect{x + pad, y + sc(4), x + w - sc(10), y + sc(26)}, cMute, f10, dtLeft)
		msg := "Fissa una build in Hellrise (pulsante 📌 nella build)"
		switch {
		case perr != "":
			msg = "Paragon non disponibile: " + perr
		case o.Planner != "" && b == nil:
			msg = "Carico il paragon…"
		case b != nil:
			msg = "Questa variante non ha un paragon"
		}
		ovText(hdc, msg, rect{x + pad, y + sc(30), x + w - sc(10), y + sc(62)}, cInk, f12, dtLeft)
		return rect{x, y, x + w, y + h}
	}
	bi := clampInt(o.Board, 0, len(step.Boards)-1)
	// righe di istruzioni
	type line struct {
		t string
		c uintptr
		f uintptr
	}
	var lines []line
	rotTxt := rotPhrase(cb.rot)
	if bi == 0 {
		rotTxt = "tavola di partenza"
	}
	lines = append(lines, line{"↻ " + rotTxt, cAcc2, ovFont(int(12.5*k), true)})
	if g := pick(board.Glyph, board.GlyphEn); g != "" {
		t := "◆ Glifo: " + g
		if board.GlyphLvl > 0 {
			t += fmt.Sprintf(" (liv. %d)", board.GlyphLvl)
		}
		lines = append(lines, line{t, col(0xC07BFF), f12})
	}
	for _, l := range board.Legend {
		p := strings.SplitN(l, "|", 2)
		en := ""
		if len(p) > 1 {
			en = p[1]
		}
		lines = append(lines, line{"★ Leggendario: " + pick(p[0], en), cAcc, f12})
		break
	}
	lines = append(lines, line{fmt.Sprintf("● %d punti su questa tavola", board.Points), cInk, f12})
	if bi+1 < len(step.Boards) {
		nb := step.Boards[bi+1]
		lines = append(lines, line{fmt.Sprintf("Poi: tavola %d · %s", bi+2, pick(nb.Name, nb.En)), cMute, f12})
	}
	head := sc(52)
	rowH := sc(20)
	foot := sc(24)
	h := head + side + sc(10) + rowH*int32(len(lines)) + foot
	x, y := widgetOrigin(o.Paragon, w, h)
	ovRound(hdc, rect{x, y, x + w, y + h}, sc(14), cPanel, cBorder)
	title := "PARAGON"
	if o.Title != "" {
		title += " · " + strings.ToUpper(o.Title)
	}
	if len(cb.variant.Paragon) > 1 {
		title += " · " + strings.ToUpper(step.Name)
	}
	ovText(hdc, title, rect{x + pad, y + sc(4), x + w - sc(70), y + sc(24)}, cMute, f10, dtLeft|0x8000)
	ovText(hdc, fmt.Sprintf("Tavola %d/%d · %s", bi+1, len(step.Boards), pick(board.Name, board.En)), rect{x + pad, y + sc(24), x + w - pad, y + sc(46)}, cInk, f13, dtLeft|0x8000)
	// mappa di tutte le tavole
	mx, my := x+pad, y+head
	ovFill(hdc, rect{mx, my, mx + side, my + side}, col(0x100A08))
	minX, maxX, minY, maxY := 1<<30, -(1 << 30), 1<<30, -(1 << 30)
	for _, bd := range step.Boards {
		minX, maxX = min(minX, bd.X), max(maxX, bd.X)
		minY, maxY = min(minY, bd.Y), max(maxY, bd.Y)
	}
	bw := board.Width
	if bw <= 0 {
		bw = 21
	}
	cols, rows := maxX-minX+1, maxY-minY+1
	span := max(cols, rows)
	unit := float64(side-sc(8)) / float64(span*(bw+1))
	ox := mx + sc(4) + int32(float64((span-cols)*(bw+1))*unit/2)
	oy := my + sc(4) + int32(float64((span-rows)*(bw+1))*unit/2)
	for i, bd := range step.Boards {
		r := bd.Rotation
		if i == bi {
			r = cb.rot
		}
		cells := rotateGrid(bd.RawCells, bd.Width, r)
		sel := rotateGrid(bd.RawSel, bd.Width, r)
		if cells == "" {
			cells, sel = bd.Cells, bd.Sel
		}
		bx := ox + int32(float64((bd.X-minX)*(bw+1))*unit)
		by := oy + int32(float64((bd.Y-minY)*(bw+1))*unit)
		drawBoardCells(hdc, cells, sel, bd.Width, bx, by, unit, 0)
		bs := int32(float64(bd.Width) * unit)
		if i == bi {
			ovFrame(hdc, rect{bx - 2, by - 2, bx + bs + 2, by + bs + 2}, 2, cAcc2)
		}
		ovText(hdc, fmt.Sprintf("%d", i+1), rect{bx, by, bx + bs, by + bs}, cInk, ovFont(int(11*k), true), dtCenter)
	}
	// la tavola di adesso, grande e già girata come va messa
	gx := mx + side + pad
	cs := float64(side) / float64(bw)
	gap := int32(1)
	if cs < 7 {
		gap = 0
	}
	drawBoardCells(hdc, cb.cells, cb.sel, bw, gx, my, cs, gap)
	ly := my + side + sc(10)
	for _, l := range lines {
		ovText(hdc, l.t, rect{x + pad, ly, x + w - pad, ly + rowH}, l.c, l.f, dtLeft|0x8000)
		ly += rowH
	}
	hint := "Ctrl+Shift+←/→ tavola · ↑/↓ fase · R/Q ruota"
	if o.Mode != "panel" {
		hint += " · apri il paragon in gioco: si allinea da solo"
	}
	ovText(hdc, hint, rect{x + pad, y + h - foot, x + w - pad, y + h - sc(4)}, cMute, ovFont(int(10*k), false), dtLeft|0x8000)
	return rect{x, y, x + w, y + h}
}

func rotLabel(r int) string {
	switch ((r % 4) + 4) % 4 {
	case 1:
		return "Maxroll: 1 scatto (90°)"
	case 2:
		return "Maxroll: 2 scatti (180°)"
	case 3:
		return "Maxroll: 3 scatti (270°)"
	}
	return "nessuna rotazione"
}

// cornice vuota (per segnare le caselle senza coprire quelle del gioco)
// anello attorno a un nodo (le caselle da prendere), più pulito delle cornici quadrate
func ovRing(hdc uintptr, r rect, t int32, color uintptr) {
	pen, _, _ := pCreatePen.Call(0, uintptr(t), color)
	op, _, _ := pSelectObject.Call(hdc, pen)
	nb, _, _ := pGetStockObject.Call(5) // NULL_BRUSH
	ob, _, _ := pSelectObject.Call(hdc, nb)
	pEllipse.Call(hdc, uintptr(r.l), uintptr(r.t), uintptr(r.r), uintptr(r.b))
	pSelectObject.Call(hdc, ob)
	pSelectObject.Call(hdc, op)
	pDeleteObject.Call(pen)
}

func ovFrame(hdc uintptr, r rect, t int32, color uintptr) {
	ovFill(hdc, rect{r.l, r.t, r.r, r.t + t}, color)
	ovFill(hdc, rect{r.l, r.b - t, r.r, r.b}, color)
	ovFill(hdc, rect{r.l, r.t, r.l + t, r.b}, color)
	ovFill(hdc, rect{r.r - t, r.t, r.r, r.b}, color)
}

// automatico: cornici sulle caselle da prendere, sopra la tavola riconosciuta sullo schermo.
// Se la schermata del paragon non è aperta non disegna nulla.
func drawParagonAligned(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	det, fresh := lastDetect()
	cb, _ := currentBoard(o)
	if !fresh || cb.step == nil || det.Board < 0 || det.Board >= len(cb.step.Boards) {
		return rect{}
	}
	cb.board = &cb.step.Boards[det.Board]
	w := cb.board.Width
	cells := rotateGrid(cb.board.RawCells, w, det.Rot)
	sel := rotateGrid(cb.board.RawSel, w, det.Rot)
	cp := det.Cell
	gx, gy := int32(det.X), int32(det.Y)
	side := int32(float64(w) * cp)
	lw, lh := sc(420), sc(28)
	lx, ly := gx, gy-lh-sc(8)
	if ly < 0 {
		ly = gy + side + sc(8)
	}
	label := fmt.Sprintf("Tavola %d/%d · %s", det.Board+1, len(cb.step.Boards), pick(cb.board.Name, cb.board.En))
	if g := pick(cb.board.Glyph, cb.board.GlyphEn); g != "" {
		label += " · glifo " + g
	}
	warn := ""
	if d := ((cb.board.Rotation-det.Rot)%4 + 4) % 4; d != 0 && det.Board > 0 {
		warn = "⚠ Nella build è girata diversamente: " + rotPhrase(d) + " prima di attaccarla"
	}
	if warn != "" {
		lh = sc(50)
		if ly < gy {
			ly = gy - lh - sc(8)
			if ly < 0 {
				ly = gy + side + sc(8)
			}
		}
		lw = sc(520)
	}
	ovLabel = rect{lx, ly, lx + lw, ly + lh}
	ovRound(hdc, rect{lx, ly, lx + lw, ly + lh}, sc(12), cPanel, cBorder)
	ovText(hdc, label, rect{lx + sc(10), ly, lx + lw - sc(10), ly + sc(28)}, cInk, ovFont(int(12*k), true), dtLeft|0x8000)
	if warn != "" {
		ovText(hdc, warn, rect{lx + sc(10), ly + sc(24), lx + lw - sc(10), ly + lh - sc(4)}, cAcc2, ovFont(int(12*k), true), dtLeft|0x8000)
	}
	t := int32(cp / 10)
	if t < 3 {
		t = 3
	}
	frames := func(cells, sel string, w int, ox, oy float64) {
		for i := 0; i < len(cells) && i < len(sel); i++ {
			if cells[i] == '0' || sel[i] != '1' {
				continue
			}
			x0 := int32(ox + float64(i%w)*cp)
			y0 := int32(oy + float64(i/w)*cp)
			x1 := int32(ox + float64(i%w+1)*cp)
			y1 := int32(oy + float64(i/w+1)*cp)
			// solo dentro la zona della mappa del paragon: non sopra le schede in alto, la barra in basso
			// e il pannello di statistiche e glifi a sinistra
			if y0 < ovH*7/100 || y1 > ovH*94/100 || x0 < ovW*20/100 || x1 > ovW {
				continue
			}
			ovRing(hdc, rect{x0 + 1, y0 + 1, x1 - 1, y1 - 1}, t, cellCol[cells[i]][1])
		}
	}
	frames(cells, sel, w, det.X, det.Y)
	// le altre tavole della build, attaccate a quella riconosciuta (ogni tavola è larga 21 caselle):
	// così con lo zoom al minimo vedi tutto il paragon
	anchor := cb.step.Boards[det.Board]
	for i, bd := range cb.step.Boards {
		if i == det.Board || bd.Width <= 0 {
			continue
		}
		ox := det.X + float64((bd.X-anchor.X)*bd.Width)*cp
		oy := det.Y + float64((bd.Y-anchor.Y)*bd.Width)*cp
		side := float64(bd.Width) * cp
		if ox+side < 0 || oy+side < 0 || ox > float64(ovW) || oy > float64(ovH) {
			continue
		}
		frames(rotateGrid(bd.RawCells, bd.Width, bd.Rotation), rotateGrid(bd.RawSel, bd.Width, bd.Rotation), bd.Width, ox, oy)
		tag := fmt.Sprintf("%d · %s", i+1, pick(bd.Name, bd.En))
		if r := rotPhrase(bd.Rotation); bd.Rotation != 0 {
			tag += " · " + r
		}
		tx, ty := int32(ox+cp*0.3), int32(oy+cp*0.3) // nell'angolo in alto a sinistra, dove la tavola è vuota
		tr := rect{tx, ty, tx + sc(250), ty + sc(24)}
		if tr.t > ovH*7/100 && tr.b < ovH*94/100 && tr.l > ovW*20/100 {
			ovRound(hdc, tr, sc(10), cPanel, cBorder)
			ovText(hdc, tag, tr, cInk, ovFont(int(11*k), true), dtCenter)
			ovTags = append(ovTags, tr)
		}
	}
	return rect{}
}

// ---------- riconoscimento automatico della tavola ----------

var det struct {
	sync.Mutex
	last BoardDetect
	at   time.Time
	cand BoardDetect
}

func lastDetect() (BoardDetect, bool) {
	det.Lock()
	defer det.Unlock()
	return det.last, det.last.OK && time.Since(det.at) < 5*time.Second
}

func autoDetectLoop() {
	for {
		det.Lock()
		locked := det.last.OK && time.Since(det.at) < 5*time.Second
		lastP := det.last.Cell
		det.Unlock()
		wait := 2200 * time.Millisecond // ricerca completa: più rada per non pesare sul gioco
		if locked {
			wait = 900 * time.Millisecond // tavola già trovata: si segue (veloce)
		}
		for t0 := time.Now(); time.Since(t0) < wait; {
			time.Sleep(100 * time.Millisecond)
			if ovMoveHeld() && !locked { // tieni premuto il tasto (Alt): cerca subito
				break
			}
		}
		o := getSettings().Overlay
		if !o.Enabled || !o.Paragon.On || o.ParMin || o.Mode == "panel" || ovHideAll || !diabloForeground() {
			if time.Since(detLogAt) > 60*time.Second && o.Enabled {
				detLogAt = time.Now()
				log.Printf("paragon: riconoscimento fermo (attivo %v, paragon %v, chiuso %v, modo %q, nascosto %v, gioco davanti %v)", o.Enabled, o.Paragon.On, o.ParMin, o.Mode, ovHideAll, diabloForeground())
			}
			continue
		}
		cb, _ := currentBoard(o)
		if cb.board == nil || cb.step == nil {
			if time.Since(detLogAt) > 60*time.Second {
				detLogAt = time.Now()
				log.Printf("paragon: nessuna tavola della build fissata")
			}
			continue
		}
		g, w, h := captureGray(rect{0, 0, ovW, ovH})
		if g == nil {
			continue
		}
		// i riquadri dell'overlay (se si vedono nella cattura) non devono confondere il riconoscimento
		ovDrawnMu.Lock()
		for _, r := range ovDrawn {
			for y := max(int(r.t), 0); y < min(int(r.b), h); y++ {
				for x := max(int(r.l), 0); x < min(int(r.r), w); x++ {
					g[y*w+x] = 20
				}
			}
		}
		ovDrawnMu.Unlock()
		f := 1.0
		if w > 1300 { // metà risoluzione basta e va molto più veloce
			g, w, h = halfGray(g, w, h)
			f = 2
		}
		// se lo schermo non è cambiato (non hai spostato né zoomato la mappa) la tavola resta dov'è:
		// niente nuova ricerca, così le cornici non ballano
		th := thumb(g, w, h)
		if locked && thumbDiff(th, lastThumb) < 2.5 {
			det.Lock()
			det.at = time.Now()
			det.Unlock()
			continue
		}
		lastThumb = th
		minP, maxP := max(10, h*12/1080), h*140/1080
		if locked {
			minP, maxP = int(lastP/f*0.95), int(lastP/f*1.05+1)
		}
		grids := findGrids(g, w, h, minP, maxP)
		// prova tutte le tavole della fase: capisce da solo quale hai aperto (anche solo in parte)
		var d BoardDetect
		bi := -1
		for i, b := range cb.step.Boards {
			for _, gr := range grids {
				if x := matchBoard(gr, b.RawCells, b.Width); x.OK && (!d.OK || x.Score > d.Score) {
					d, bi = x, i
				}
			}
		}
		bestScore := d.Score
		if !d.OK {
			for _, b := range cb.step.Boards {
				for _, gr := range grids {
					if x := matchBoard(gr, b.RawCells, b.Width); x.Score > bestScore {
						bestScore = x.Score
					}
				}
			}
		}
		if d.OK {
			d = refineDetect(g, w, h, d, cb.step.Boards[bi].RawCells, cb.step.Boards[bi].Width)
		}
		if time.Since(detLogAt) > 20*time.Second {
			detLogAt = time.Now()
			var lum float64
			for _, v := range g {
				lum += v
			}
			log.Printf("paragon: cattura %dx%d (luce media %.0f), %d griglie, tavola %d trovata=%v punteggio %.2f, casella %.1f", w, h, lum/float64(len(g)), len(grids), bi, d.OK, bestScore, d.Cell)
		}
		d.Board = bi
		det.Lock()
		if d.OK {
			d.X, d.Y, d.Cell = d.X*f, d.Y*f, d.Cell*f
			c := det.cand
			near := func(a BoardDetect) bool {
				return a.OK && absf(a.Cell-d.Cell) < 0.05*d.Cell && absf(a.X-d.X) < d.Cell/3 && absf(a.Y-d.Y) < d.Cell/3 && a.Rot == d.Rot && a.Board == d.Board
			}
			same := near(c)
			if near(det.last) && time.Since(det.at) < 6*time.Second {
				// stessa tavola di prima: le cornici restano ferme (niente tremolio), si sposta solo
				// se lo schermo si è mosso davvero
				l := det.last
				if absf(l.X-d.X) < d.Cell*0.12 && absf(l.Y-d.Y) < d.Cell*0.12 && absf(l.Cell-d.Cell) < 0.02*d.Cell {
					d.X, d.Y, d.Cell = l.X, l.Y, l.Cell
				}
			}
			if same || d.Score >= 0.88 || (det.last.OK && time.Since(det.at) < 6*time.Second) {
				det.last, det.at = d, time.Now()
				if bi >= 0 && bi != o.Board { // la guida segue la tavola che hai aperto in gioco
					go updateSettings(func(s *Settings) { s.Overlay.Board = bi })
				}
			}
			det.cand = d
		} else {
			det.cand = BoardDetect{}
		}
		det.Unlock()
	}
}

var detLogAt time.Time

var lastThumb []float64

// immagine piccolissima dello schermo (64x36) per capire se è cambiato
func thumb(g []float64, w, h int) []float64 {
	const tw, thh = 64, 36
	out := make([]float64, tw*thh)
	for y := 0; y < thh; y++ {
		for x := 0; x < tw; x++ {
			var s float64
			n := 0
			for yy := y * h / thh; yy < (y+1)*h/thh; yy += 2 {
				for xx := x * w / tw; xx < (x+1)*w/tw; xx += 2 {
					s += g[yy*w+xx]
					n++
				}
			}
			if n > 0 {
				out[y*tw+x] = s / float64(n)
			}
		}
	}
	return out
}

func thumbDiff(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1e9
	}
	var s float64
	for i := range a {
		s += absf(a[i] - b[i])
	}
	return s / float64(len(a))
}

func ovMoveHeld() bool { return ovMove }

// zone dove l'overlay ha disegnato i suoi riquadri (per toglierle dalla cattura)
var (
	ovDrawnMu sync.Mutex
	ovDrawn   []rect
)

// ---------- rilevamento della grandezza delle caselle ----------

// cattura una zona dello schermo in scala di grigi
func captureGray(r rect) ([]float64, int, int) {
	w, h := int(r.r-r.l), int(r.b-r.t)
	if w < 50 || h < 50 {
		return nil, 0, 0
	}
	sdc, _, _ := pGetDC.Call(0)
	defer pReleaseDC.Call(0, sdc)
	mem, _, _ := pCreateCompatibleDC.Call(sdc)
	bmp, _, _ := pCreateCompatibleBmp.Call(sdc, uintptr(w), uintptr(h))
	old, _, _ := pSelectObject.Call(mem, bmp)
	pBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), sdc, uintptr(r.l), uintptr(r.t), srcCopy)
	pSelectObject.Call(mem, old)
	type bih struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPels, YPels           int32
		ClrUsed, ClrImportant  uint32
	}
	hdr := bih{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
	buf := make([]byte, w*h*4)
	pGetDIBits.Call(mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&hdr)), 0)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mem)
	g := make([]float64, w*h)
	for i := 0; i < w*h; i++ {
		g[i] = 0.114*float64(buf[i*4]) + 0.587*float64(buf[i*4+1]) + 0.299*float64(buf[i*4+2])
	}
	return g, w, h
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
	keys := map[int]uintptr{1: 'O', 2: 'P', 3: 'T', 4: 'E', 5: 0x27, 6: 0x25, 7: 0x26, 8: 0x28,
		9: 'R', 10: 'Q', 14: 'M'}
	for id, vk := range keys {
		if on {
			pRegisterHotKey.Call(ovHwnd, uintptr(id), modControl|modShift|modNoRepeat, vk)
		} else {
			pUnregisterHotKey.Call(ovHwnd, uintptr(id))
		}
	}
	if on { // Alt+Shift+P: apre o chiude il paragon (occhietto)
		pRegisterHotKey.Call(ovHwnd, 15, 0x0001|modShift|modNoRepeat, 'P')
	} else {
		pUnregisterHotKey.Call(ovHwnd, 15)
	}
}

func setOverlayEdit(on bool) {
	ovEdit = on
	setClickable(ovEdit || ovMove)
}

// l'overlay riceve i clic solo mentre sposti i riquadri; il resto del tempo i clic vanno al gioco
func setClickable(on bool) {
	ex, _, _ := pGetWindowLongPtr.Call(ovHwnd, ^uintptr(19)) // GWL_EXSTYLE (-20)
	if on {
		ex &^= wsExTransparent
	} else {
		ex |= wsExTransparent
	}
	pSetWindowLongPtr.Call(ovHwnd, ^uintptr(19), ex)
}

// l'overlay compare in OBS e nelle registrazioni, a meno che tu non scelga di nasconderlo
var ovCapHidden = -1

func applyCaptureAffinity() {
	hide := 0
	if getSettings().Overlay.HideCap {
		hide = 1
	}
	if hide == ovCapHidden || ovHwnd == 0 {
		return
	}
	ovCapHidden = hide
	if p := u32.NewProc("SetWindowDisplayAffinity"); p.Find() == nil {
		if hide == 1 {
			p.Call(ovHwnd, 0x11) // WDA_EXCLUDEFROMCAPTURE
		} else {
			p.Call(ovHwnd, 0) // WDA_NONE
		}
	}
}

func overlayTick() {
	applyCaptureAffinity()
	o := getSettings().Overlay
	fg := diabloForeground()
	// trasparenza di tutto l'overlay
	a := clampInt(o.Opacity, 20, 100)
	if a != ovAlpha {
		ovAlpha = a
		pSetLayeredAttr.Call(ovHwnd, keyColor, uintptr(a*255/100), lwaColorKey|0x2) // COLORKEY | ALPHA
	}
	// tasto tenuto premuto: si possono spostare e bloccare i riquadri
	held := false
	if o.Enabled && o.MoveKey > 0 && (fg || o.Always || ovMove) {
		st, _, _ := pGetAsyncKeyState.Call(uintptr(o.MoveKey))
		held = st&0x8000 != 0
	}
	if held != ovMove {
		ovMove = held
		if !held && ovDrag != "" {
			ovDrag = ""
			pReleaseCapture.Call()
			saveSettings()
		}
		setClickable(ovEdit || ovMove)
	}
	show := o.Enabled && !ovHideAll && (o.Always || fg || ovEdit || ovMove)
	hot := o.Enabled && (o.Always || fg || ovEdit)
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
	ovTicks++
	_, aligned := lastDetect()
	if show && (ovEdit || ovMove || aligned || ovTicks%5 == 0) {
		pInvalidateRect.Call(ovHwnd, 0, 0)
	}
}

var ovTicks int

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
	case 9, 10:
		cb, _ := currentBoard(getSettings().Overlay)
		if cb.board != nil {
			d := 1
			if id == 10 {
				d = -1
			}
			updateSettings(func(s *Settings) {
				if s.Overlay.Rot == nil {
					s.Overlay.Rot = map[string]int{}
				}
				s.Overlay.Rot[cb.key] = ((cb.rot+d)%4 + 4) % 4
			})
		}
	case 15:
		updateSettings(func(s *Settings) { s.Overlay.ParMin = !s.Overlay.ParMin })
	case 14:
		updateSettings(func(s *Settings) {
			if s.Overlay.Mode == "panel" {
				s.Overlay.Mode = "auto"
			} else {
				s.Overlay.Mode = "panel"
			}
		})
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

func resizeCells(o *OverlaySet, grow bool, steps float64) {
	if grow {
		o.Cell++
	} else {
		o.Cell--
	}
	o.Cell = clampInt(o.Cell, 5, 40)
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
	case 0x0201: // WM_LBUTTONDOWN: lucchetto oppure inizia a trascinare un riquadro
		if ovEdit || ovMove {
			x, y := int32(int16(lparam&0xFFFF)), int32(int16(lparam>>16&0xFFFF))
			in := func(r rect) bool { return x >= r.l && x <= r.r && y >= r.t && y <= r.b }
			if ovEye.r > ovEye.l && in(ovEye) {
				updateSettings(func(s *Settings) { s.Overlay.ParMin = !s.Overlay.ParMin })
				pInvalidateRect.Call(hwnd, 0, 0)
				return 0
			}
			for key, r := range ovLocks {
				if in(r) {
					updateSettings(func(s *Settings) {
						w := widgetOf(&s.Overlay, key)
						w.Lock = !w.Lock
					})
					pInvalidateRect.Call(hwnd, 0, 0)
					return 0
				}
			}
			o := getSettings().Overlay
			for key, r := range ovHits {
				if !in(r) || widgetOf(&o, key).Lock {
					continue
				}
				ovDrag, ovDragDX, ovDragDY = key, x-r.l, y-r.t
				pSetCapture.Call(hwnd)
			}
		}
		return 0
	case 0x0200: // WM_MOUSEMOVE
		if ovDrag != "" {
			x, y := int32(int16(lparam&0xFFFF)), int32(int16(lparam>>16&0xFFFF))
			fx, fy := float64(x-ovDragDX)/float64(ovW), float64(y-ovDragDY)/float64(ovH)
			key := ovDrag
			setMu.Lock()
			w := widgetOf(&settings.Overlay, key)
			w.X, w.Y = fx, fy
			setMu.Unlock()
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case 0x020A: // WM_MOUSEWHEEL: grandezza del riquadro paragon mentre sposti
		if ovEdit || ovMove {
			delta := int16(wparam >> 16)
			updateSettings(func(s *Settings) { resizeCells(&s.Overlay, delta > 0, 1) })
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
	pSetLayeredAttr.Call(ovHwnd, keyColor, 255, lwaColorKey|0x2)
	applyCaptureAffinity()
	pSetTimer.Call(ovHwnd, 1, 100, 0)
	go autoDetectLoop()
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
