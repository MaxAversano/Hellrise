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
//   Ctrl+Shift+R / Q  ruota la tavola in senso orario / antiorario
//   Ctrl+Shift+M  riquadro ↔ sovrapposto alla tavola del gioco
//   Ctrl+Shift+D  rileva la grandezza delle caselle dallo schermo
//   Ctrl+Shift+ + / −  caselle più grandi / più piccole (anche con la rotellina mentre sposti)

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
		ovText(mem, "Trascina i riquadri · rotellina = caselle · Ctrl+Shift+E per finire", rect{ovW/2 - sc(230), sc(14), ovW/2 + sc(230), sc(50)}, cInk, ovFont(int(13*k), true), dtCenter)
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
	if o.Mode == "align" {
		return drawParagonAligned(hdc, o, sc, k)
	}
	cell := o.Cell
	if cell <= 0 {
		cell = 13
	}
	cs := sc(float64(cell))
	if cs < 4 {
		cs = 4
	}
	cb, perr := currentBoard(o)
	b, board, step, variant := cb.b, cb.board, cb.step, cb.variant
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
	rotTxt := rotLabel(board.Rotation)
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
	for i := 0; i < len(cb.cells); i++ {
		c := cb.cells[i]
		if c == '0' {
			continue
		}
		cc, ok := cellCol[c]
		if !ok {
			continue
		}
		cx, cy := gx+int32(i%bw)*cs, gy+int32(i/bw)*cs
		r := rect{cx + gap, cy + gap, cx + cs - gap, cy + cs - gap}
		if cb.sel[i] == '1' {
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
	hint := "Ctrl+Shift+←/→ tavola · R/Q ruota · M sovrapponi"
	if cb.rot != board.Rotation {
		hint = "Ruotata da te: " + rotLabel(cb.rot) + " · " + hint
	}
	ovText(hdc, hint, rect{x + sc(14), fy + sc(22), x + w - sc(14), fy + sc(42)}, cMute, ovFont(int(10.5*k), false), dtLeft)
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
func ovFrame(hdc uintptr, r rect, t int32, color uintptr) {
	ovFill(hdc, rect{r.l, r.t, r.r, r.t + t}, color)
	ovFill(hdc, rect{r.l, r.b - t, r.r, r.b}, color)
	ovFill(hdc, rect{r.l, r.t, r.l + t, r.b}, color)
	ovFill(hdc, rect{r.r - t, r.t, r.r, r.b}, color)
}

// modalità sovrapposta: solo cornici sulle caselle da prendere, allineate alla tavola del gioco
func drawParagonAligned(hdc uintptr, o OverlaySet, sc func(float64) int32, k float64) rect {
	cb, perr := currentBoard(o)
	cp := o.CellPx
	if cp < 6 {
		cp = 40
	}
	w := 21
	if cb.board != nil {
		w = cb.board.Width
	}
	gx, gy := int32(o.AlignX*float64(ovW)), int32(o.AlignY*float64(ovH))
	side := int32(float64(w) * cp)
	// etichetta sopra la griglia
	lw, lh := sc(430), sc(28)
	lx, ly := gx, gy-lh-sc(8)
	if ly < 0 {
		ly = gy + side + sc(8)
	}
	label := "Paragon: scegli una build in Hellrise → Overlay"
	if perr != "" {
		label = "Planner non disponibile: " + perr
	}
	if cb.board != nil {
		label = fmt.Sprintf("Tavola %d/%d · %s · %s", clampInt(o.Board, 0, len(cb.step.Boards)-1)+1, len(cb.step.Boards), pick(cb.board.Name, cb.board.En), rotLabel(cb.board.Rotation))
		if g := pick(cb.board.Glyph, cb.board.GlyphEn); g != "" {
			label += " · glifo " + g
		}
	}
	ovRound(hdc, rect{lx, ly, lx + lw, ly + lh}, sc(12), cPanel, cBorder)
	ovText(hdc, label, rect{lx + sc(10), ly, lx + lw - sc(10), ly + lh}, cInk, ovFont(int(12*k), true), dtLeft)
	if cb.board == nil {
		return rect{lx, ly, lx + lw, ly + lh}
	}
	t := int32(cp / 12)
	if t < 2 {
		t = 2
	}
	for i := 0; i < len(cb.cells); i++ {
		c := cb.cells[i]
		if c == '0' {
			continue
		}
		x0 := gx + int32(float64(i%w)*cp)
		y0 := gy + int32(float64(i/w)*cp)
		x1 := gx + int32(float64(i%w+1)*cp)
		y1 := gy + int32(float64(i/w+1)*cp)
		r := rect{x0 + 1, y0 + 1, x1 - 1, y1 - 1}
		if cb.sel[i] == '1' {
			cc := cellCol[c]
			ovFrame(hdc, r, t, cc[1])
		} else if ovEdit {
			ovFrame(hdc, r, 1, col(0x5A4A40)) // in modalità "sposta" si vede tutta la griglia per allinearla
		}
	}
	return rect{gx, gy, gx + side, gy + side}
}

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

func detectCellSize() {
	o := getSettings().Overlay
	var area rect
	if o.Mode == "align" && o.CellPx >= 6 {
		side := int32(21 * o.CellPx)
		gx, gy := int32(o.AlignX*float64(ovW)), int32(o.AlignY*float64(ovH))
		area = rect{gx - side/5, gy - side/5, gx + side + side/5, gy + side + side/5}
	} else {
		area = rect{ovW / 5, ovH / 8, ovW * 4 / 5, ovH * 7 / 8}
	}
	if area.l < 0 {
		area.l = 0
	}
	if area.t < 0 {
		area.t = 0
	}
	if area.r > ovW {
		area.r = ovW
	}
	if area.b > ovH {
		area.b = ovH
	}
	// nasconde l'overlay per un attimo, così nella cattura c'è solo il gioco
	pShowWindow.Call(ovHwnd, 0)
	time.Sleep(120 * time.Millisecond)
	g, w, h := captureGray(area)
	if ovVisible {
		pShowWindow.Call(ovHwnd, 8)
	}
	if g == nil {
		return
	}
	cols := make([]float64, w-1)
	rows := make([]float64, h-1)
	for y := 0; y < h; y++ {
		for x := 0; x < w-1; x++ {
			d := g[y*w+x+1] - g[y*w+x]
			if d < 0 {
				d = -d
			}
			cols[x] += d
		}
	}
	for y := 0; y < h-1; y++ {
		for x := 0; x < w; x++ {
			d := g[(y+1)*w+x] - g[y*w+x]
			if d < 0 {
				d = -d
			}
			rows[y] += d
		}
	}
	minL, maxL := int(float64(ovH)/1080*14), int(float64(ovH)/1080*130)
	px, rx := dominantPeriod(cols, minL, maxL)
	py, ry := dominantPeriod(rows, minL, maxL)
	var cell float64
	switch {
	case px > 0 && py > 0 && absf(float64(px-py)) <= 0.15*float64(px):
		cell = float64(px+py) / 2
	case rx >= ry && px > 0:
		cell = float64(px)
	case py > 0:
		cell = float64(py)
	}
	log.Printf("overlay: caselle rilevate x=%d (%.2f) y=%d (%.2f) → %.1f px", px, rx, py, ry, cell)
	if cell < 6 || (rx < 0.15 && ry < 0.15) {
		notify("Paragon: caselle non riconosciute", "Apri la schermata del paragon, centra la tavola e riprova (Ctrl+Shift+D)")
		return
	}
	updateSettings(func(s *Settings) {
		s.Overlay.CellPx = cell
		s.Overlay.Mode = "align"
	})
	notify("Paragon: caselle riconosciute", fmt.Sprintf("Passo di %.0f pixel. Con Ctrl+Shift+E sposta la griglia sopra la tavola.", cell))
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
		9: 'R', 10: 'Q', 11: 0xBB, 12: 0xBD, 13: 'D', 14: 'M', 15: 0x6B, 16: 0x6D}
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
	case 11, 12, 15, 16:
		grow := id == 11 || id == 15
		updateSettings(func(s *Settings) { resizeCells(&s.Overlay, grow, 1) })
	case 13:
		go detectCellSize()
	case 14:
		updateSettings(func(s *Settings) {
			if s.Overlay.Mode == "align" {
				s.Overlay.Mode = "panel"
			} else {
				s.Overlay.Mode = "align"
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
	if o.Mode == "align" {
		d := 0.5 * steps
		if !grow {
			d = -d
		}
		o.CellPx = float64(int((o.CellPx+d)*2+0.5)) / 2
		if o.CellPx < 6 {
			o.CellPx = 6
		}
		if o.CellPx > 160 {
			o.CellPx = 160
		}
		return
	}
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
			} else if settings.Overlay.Mode == "align" {
				settings.Overlay.AlignX, settings.Overlay.AlignY = fx, fy
			} else {
				settings.Overlay.Paragon.X, settings.Overlay.Paragon.Y = fx, fy
			}
			setMu.Unlock()
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case 0x020A: // WM_MOUSEWHEEL: grandezza delle caselle mentre sposti
		if ovEdit {
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
