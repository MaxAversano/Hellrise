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
	Cells    string   `json:"cells"` // una cifra per casella, già ruotata
	Sel      string   `json:"sel"`   // "1" = da prendere
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
	for i := range cells {
		cells[i], sel[i] = '0', '0'
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
		out.Points++
		if i < len(def.Cells) && def.Cells[i] == 4 && d != nil {
			nn := d.NodeNames[def.Names[i]]
			if nn[0] != "" || nn[1] != "" {
				out.Legend = append(out.Legend, nn[0]+"|"+nn[1])
			}
		}
	}
	out.Width, out.Cells, out.Sel = w, string(cells), string(sel)
	return out
}
