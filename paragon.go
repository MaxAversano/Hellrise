package main

// Paragon dai planner Maxroll: tavole nell'ordine del planner, ruotate come in gioco,
// con le caselle da prendere. Usate dalla pagina Build e dall'overlay in gioco.

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type rawBoard struct {
	ID       string  `json:"id"`
	Nodes    nodeSet `json:"nodes"`
	Rotation int     `json:"rotation"`
	Glyph    string  `json:"glyph"`
	GlyphLvl int     `json:"glyphLevel"`
	Position struct {
		X, Y int
	} `json:"position"`
}

// nodi presi: di solito {"123":1}, ma vuoto arriva come [] (e a volte come lista di indici)
type nodeSet map[string]int

func (n *nodeSet) UnmarshalJSON(b []byte) error {
	m := map[string]int{}
	if json.Unmarshal(b, &m) == nil {
		*n = m
		return nil
	}
	var l []any
	if json.Unmarshal(b, &l) == nil {
		for _, x := range l {
			if f, ok := x.(float64); ok {
				m[strconv.Itoa(int(f))] = 1
			}
		}
	}
	*n = m
	return nil
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
	Keys     []string `json:"keys,omitempty"`   // id dei nodi, casella per casella, senza rotazione
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
	out.Keys = def.Names
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

// ---------- riconoscimento automatico della tavola sullo schermo ----------

type BoardDetect struct {
	OK    bool
	X, Y  float64 // angolo in alto a sinistra della griglia (pixel dell'immagine)
	Cell  float64 // passo delle caselle in pixel
	Rot   int     // rotazione della tavola vista sullo schermo (quarti di giro rispetto ai dati)
	Score float64
	Board int // quale tavola della build (riconoscimento automatico)
}

// cerca nella schermata (scala di grigi) una griglia regolare di caselle e la tavola che combacia meglio.
// raw: caselle della tavola senza rotazione ('0' = vuota); bw: lato della tavola.
func detectBoard(g []float64, w, h int, raw string, bw int, minP, maxP int) BoardDetect {
	var d BoardDetect
	if len(g) != w*h || w < 100 || h < 100 || bw <= 0 || len(raw) != bw*bw {
		return d
	}
	// 1) passo della griglia dai bordi verticali e orizzontali
	cols := make([]float64, w-1)
	rows := make([]float64, h-1)
	for y := 0; y < h-1; y++ {
		o := y * w
		for x := 0; x < w-1; x++ {
			cols[x] += absf(g[o+x+1] - g[o+x])
			rows[y] += absf(g[o+w+x] - g[o+x])
		}
	}
	px, rx := dominantPeriod(cols, minP, maxP)
	py, ry := dominantPeriod(rows, minP, maxP)
	// se i due assi non concordano, cerca nell'altro asse attorno al periodo più forte
	if px > 0 && py > 0 && absf(float64(px-py)) > 0.12*float64(px) {
		if rx >= ry {
			if q, rq := dominantPeriod(rows, px-3, px+3); q > 0 && rq > 0.2 {
				py, ry = q, maxf(rq, 0.25)
			}
		} else if q, rq := dominantPeriod(cols, py-3, py+3); q > 0 && rq > 0.2 {
			px, rx = q, maxf(rq, 0.25)
		}
	}
	if px == 0 || py == 0 || rx < 0.25 || ry < 0.25 || absf(float64(px-py)) > 0.12*float64(px) {
		return d
	}
	P := float64(px+py) / 2
	// 2) fase: le righe/colonne più scure tra una casella e l'altra
	phase := func(prof []float64) float64 {
		n := int(P + 0.5)
		best, bestV := 0, 1e18
		for p := 0; p < n; p++ {
			var s float64
			c := 0
			for x := float64(p); int(x) < len(prof); x += P {
				s += prof[int(x)]
				c++
			}
			if c > 0 && s/float64(c) < bestV {
				best, bestV = p, s/float64(c)
			}
		}
		return float64(best)
	}
	colMean := make([]float64, w)
	rowMean := make([]float64, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := g[y*w+x]
			colMean[x] += v
			rowMean[y] += v
		}
	}
	bx, by := phase(colMean), phase(rowMean)
	nx, ny := int((float64(w)-bx)/P), int((float64(h)-by)/P)
	if nx < 6 || ny < 6 {
		return d
	}
	// 3) presenza di un nodo in ogni casella (energia dei bordi nella parte centrale)
	E := make([]float64, nx*ny)
	vals := make([]float64, 0, nx*ny)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			x0, y0 := int(bx+float64(i)*P+P*0.22), int(by+float64(j)*P+P*0.22)
			x1, y1 := int(bx+float64(i)*P+P*0.78), int(by+float64(j)*P+P*0.78)
			var s float64
			c := 0
			for y := y0; y < y1 && y < h-1; y++ {
				for x := x0; x < x1 && x < w-1; x++ {
					s += absf(g[y*w+x+1]-g[y*w+x]) + absf(g[(y+1)*w+x]-g[y*w+x])
					c++
				}
			}
			if c > 0 {
				E[j*nx+i] = s / float64(c)
			}
			vals = append(vals, E[j*nx+i])
		}
	}
	// soglia a due gruppi (k-means a 2)
	lo, hi := 1e18, -1e18
	for _, v := range vals {
		lo, hi = minf(lo, v), maxf(hi, v)
	}
	if hi-lo < 1 {
		return d
	}
	thr := (lo + hi) / 2
	for it := 0; it < 12; it++ {
		var a, b float64
		var na, nb int
		for _, v := range vals {
			if v > thr {
				a += v
				na++
			} else {
				b += v
				nb++
			}
		}
		if na == 0 || nb == 0 {
			break
		}
		thr = (a/float64(na) + b/float64(nb)) / 2
	}
	pres := make([]bool, nx*ny)
	for i, v := range E {
		pres[i] = v > thr
	}
	// 4) la tavola (in ognuna delle 4 rotazioni) che combacia meglio con le caselle piene
	bestScore, bestR, bestDX, bestDY := -1e9, 0, 0, 0
	for r := 0; r < 4; r++ {
		m := rotateGrid(raw, bw, r)
		total := 0
		for i := 0; i < len(m); i++ {
			if m[i] != '0' {
				total++
			}
		}
		if total == 0 {
			continue
		}
		for dy := -3; dy <= ny-bw+3; dy++ {
			for dx := -3; dx <= nx-bw+3; dx++ {
				var s float64
				inView := 0
				for j := 0; j < bw; j++ {
					yy := dy + j
					if yy < 0 || yy >= ny {
						continue
					}
					for i := 0; i < bw; i++ {
						xx := dx + i
						if xx < 0 || xx >= nx {
							continue
						}
						p := pres[yy*nx+xx]
						if m[j*bw+i] != '0' {
							inView++
							if p {
								s++
							} else {
								s -= 0.6
							}
						} else if p {
							s -= 0.4
						}
					}
				}
				if float64(inView) < 0.7*float64(total) {
					continue
				}
				sc := s / float64(total)
				if sc > bestScore {
					bestScore, bestR, bestDX, bestDY = sc, r, dx, dy
				}
			}
		}
	}
	if bestScore < 0.55 {
		d.Score = bestScore
		return d
	}
	return BoardDetect{OK: true, X: bx + float64(bestDX)*P, Y: by + float64(bestDY)*P, Cell: P, Rot: bestR, Score: bestScore}
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// paragon di una build D4Builds nel formato dell'overlay: le tavole vengono riconosciute
// sui dati di gioco (stessa forma delle tavole Maxroll) e i nodi presi presi dalla pagina
func d4bParagonBuild(u string) (*PBuild, error) {
	var f D4Full
	if b := serverFull(u); b != nil {
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, err
		}
	} else if isCollector() {
		ff, err := d4bFullRead(u)
		if err != nil {
			return nil, err
		}
		f = *ff
	} else {
		return nil, fmt.Errorf("build non ancora raccolta")
	}
	d := getGD()
	cls := strings.ToLower(f.Head["cls"])
	tok := map[string]string{"barbarian": "barb", "druid": "druid", "necromancer": "necro", "rogue": "rogue", "sorcerer": "sorc", "spiritborn": "spirit", "paladin": "paladin", "warlock": "warlock"}[cls]
	boardID := func(name string) string {
		if d == nil {
			return ""
		}
		if strings.HasPrefix(strings.ToLower(name), "starting") {
			name = "Start"
		}
		var cands []string
		for k, b := range d.Boards {
			if strings.EqualFold(b.En, name) {
				cands = append(cands, k)
			}
		}
		sort.Strings(cands)
		for _, k := range cands {
			if tok != "" && strings.Contains(strings.ToLower(k), tok) {
				return k
			}
		}
		if len(cands) > 0 {
			return cands[0]
		}
		return ""
	}
	glyphID := func(name string) string {
		if d == nil || name == "" {
			return ""
		}
		for k, g := range d.Glyphs {
			if strings.EqualFold(g[1], name) {
				return k
			}
		}
		return ""
	}
	out := &PBuild{ID: "d4b", Name: f.Head["title"], URL: u}
	for _, v := range f.Variants {
		pv := PVariant{Name: fmt.Sprint(v["name"])}
		par, _ := v["paragon"].(map[string]any)
		boards, _ := par["boards"].([]any)
		st := PStep{Name: "Paragon"}
		for _, bx := range boards {
			bm, _ := bx.(map[string]any)
			name, _ := bm["n"].(string)
			g, _ := bm["g"].(string)
			rot, _ := bm["r"].(float64)
			id := boardID(name)
			rb := rawBoard{ID: id, Nodes: map[string]int{}, Rotation: (int(rot)/90%4 + 4) % 4, Glyph: glyphID(g)}
			px, _ := bm["x"].(float64)
			py, _ := bm["y"].(float64)
			rb.Position.X, rb.Position.Y = int(math.Round(px/1260)), int(math.Round(py/1260))
			w := 21
			if d != nil && d.Boards[id].Width > 0 {
				w = d.Boards[id].Width
			}
			tiles, _ := bm["tiles"].([]any)
			for _, tx := range tiles {
				t, _ := tx.([]any)
				if len(t) < 3 {
					continue
				}
				r, _ := t[0].(float64)
				c, _ := t[1].(float64)
				a, _ := t[2].(float64)
				if a == 1 {
					rb.Nodes[itoa(int(r)*w+int(c))] = 1
				}
			}
			pb := paragonBoard(d, rb)
			if id == "" {
				pb.Name, pb.En = name, name
			}
			st.Boards = append(st.Boards, pb)
		}
		if len(st.Boards) > 0 {
			pv.Paragon = []PStep{st}
		}
		out.Variants = append(out.Variants, pv)
	}
	return out, nil
}
