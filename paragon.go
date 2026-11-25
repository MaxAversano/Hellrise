package main

// Paragon dai planner Maxroll: tavole nell'ordine del planner, ruotate come in gioco,
// con le caselle da prendere. Usate dalla pagina Build e dall'overlay in gioco.

import (
	"sort"
	"strconv"
)

type rawBoard struct {
	ID       string         `json:"id"`
	Nodes    map[string]int `json:"nodes"`
	Rotation int            `json:"rotation"`
	Glyph    string         `json:"glyph"`
	GlyphLvl int            `json:"glyphLevel"`
	Position struct {
		X, Y int
	} `json:"position"`
}

type PBoard struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	En       string   `json:"en"`
	Rotation int      `json:"rotation"` // in quarti di giro (0-3)
	Glyph    string   `json:"glyph,omitempty"`
	GlyphEn  string   `json:"glyphEn,omitempty"`
	GlyphLvl int      `json:"glyphLvl,omitempty"`
	Width    int      `json:"width"`
	Cells    string   `json:"cells"`    // una cifra per casella, già ruotata
	Sel      string   `json:"sel"`      // "1" = da prendere
	RawCells string   `json:"rawCells"` // come sopra, senza rotazione (l'overlay la gira a piacere)
	RawSel   string   `json:"rawSel"`
	Points   int      `json:"points"`
	Legend   []string `json:"legend,omitempty"` // nodi leggendari presi
	X        int      `json:"x"`
	Y        int      `json:"y"`
}

type PStep struct {
	Name   string   `json:"name"`
	Boards []PBoard `json:"boards"`
}

func paragonBoard(d *gameData, b rawBoard) PBoard {
	out := PBoard{ID: b.ID, Rotation: ((b.Rotation % 4) + 4) % 4, GlyphLvl: b.GlyphLvl, X: b.Position.X, Y: b.Position.Y, Name: prettyKey(b.ID)}
	var def PBoardDef
	if d != nil {
		def = d.Boards[b.ID]
		if def.It != "" || def.En != "" {
			out.Name, out.En = def.It, def.En
		}
		if b.Glyph != "" {
			g := d.Glyphs[b.Glyph]
			out.Glyph, out.GlyphEn = g[0], g[1]
		}
	}
	if b.Glyph != "" && out.Glyph == "" {
		out.Glyph = prettyKey(b.Glyph)
	}
	w := def.Width
	if w == 0 {
		w = 21
	}
	n := w * w
	cells := make([]byte, n)
	sel := make([]byte, n)
	rc := make([]byte, n)
	rs := make([]byte, n)
	for i := range cells {
		cells[i], sel[i], rc[i], rs[i] = '0', '0', '0', '0'
	}
	rot := func(i int) int { // rotazione in senso orario di r quarti
		x, y := i%w, i/w
		for k := 0; k < out.Rotation; k++ {
			x, y = w-1-y, x
		}
		return y*w + x
	}
	for i := 0; i < n && i < len(def.Cells); i++ {
		cells[rot(i)] = '0' + def.Cells[i]
		rc[i] = '0' + def.Cells[i]
	}
	keys := make([]int, 0, len(b.Nodes))
	for k := range b.Nodes {
		if i, err := strconv.Atoi(k); err == nil && i >= 0 && i < n {
			keys = append(keys, i)
		}
	}
	sort.Ints(keys)
	for _, i := range keys {
		sel[rot(i)] = '1'
		rs[i] = '1'
		out.Points++
		if i < len(def.Cells) && def.Cells[i] == 4 && d != nil {
			nn := d.NodeNames[def.Names[i]]
			if nn[0] != "" || nn[1] != "" {
				out.Legend = append(out.Legend, nn[0]+"|"+nn[1])
			}
		}
	}
	out.Width, out.Cells, out.Sel, out.RawCells, out.RawSel = w, string(cells), string(sel), string(rc), string(rs)
	return out
}

// ruota una griglia quadrata di r quarti di giro in senso orario
func rotateGrid(g string, w, r int) string {
	r = ((r % 4) + 4) % 4
	if r == 0 || w <= 0 || len(g) != w*w {
		return g
	}
	out := make([]byte, len(g))
	for i := 0; i < len(g); i++ {
		x, y := i%w, i/w
		for k := 0; k < r; k++ {
			x, y = w-1-y, x
		}
		out[y*w+x] = g[i]
	}
	return string(out)
}

// periodo dominante di un segnale (autocorrelazione), con la forza del picco
func dominantPeriod(sig []float64, minL, maxL int) (int, float64) {
	n := len(sig)
	if maxL > n/3 {
		maxL = n / 3
	}
	mean := 0.0
	for _, v := range sig {
		mean += v
	}
	mean /= float64(n)
	x := make([]float64, n)
	var e0 float64
	for i, v := range sig {
		x[i] = v - mean
		e0 += x[i] * x[i]
	}
	if e0 == 0 || maxL <= minL {
		return 0, 0
	}
	best, bestR := 0, 0.0
	rs := make([]float64, maxL+1)
	for l := minL; l <= maxL; l++ {
		var s float64
		for i := 0; i+l < n; i++ {
			s += x[i] * x[i+l]
		}
		rs[l] = s / e0 * float64(n) / float64(n-l)
		if rs[l] > bestR {
			best, bestR = l, rs[l]
		}
	}
	if best == 0 {
		return 0, 0
	}
	// preferisce il periodo più piccolo con un picco quasi alto quanto il migliore (evita i multipli)
	for l := minL + 1; l < best; l++ {
		if rs[l] >= 0.85*bestR && rs[l] >= rs[l-1] && rs[l] >= rs[l+1] {
			return l, rs[l]
		}
	}
	return best, bestR
}

func absf(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
