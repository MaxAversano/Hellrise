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
var detDebug func(P, bx, by float64, nx, ny int, pres []bool)

// griglia di caselle trovata sullo schermo
type PGrid struct {
	P, X, Y float64
	NX, NY  int
	Pres    []bool
}

func detectBoard(g []float64, w, h int, raw string, bw int, minP, maxP int) BoardDetect {
	var best BoardDetect
	if len(g) != w*h || w < 100 || h < 100 || bw <= 0 || len(raw) != bw*bw {
		return best
	}
	for _, gr := range findGrids(g, w, h, minP, maxP) {
		if x := matchBoard(gr, raw, bw); x.OK && (!best.OK || x.Score > best.Score) {
			best = x
		} else if !best.OK && x.Score > best.Score {
			best.Score = x.Score
		}
	}
	return refineDetect(g, w, h, best, raw, bw)
}

// la tavola (in ognuna delle 4 rotazioni) che combacia meglio con le caselle piene della griglia.
// Basta vederne una parte (chi zooma vede solo metà tavola).
func matchBoard(gr PGrid, raw string, bw int) BoardDetect {
	var d BoardDetect
	P, nx, ny, pres := gr.P, gr.NX, gr.NY, gr.Pres
	if P == 0 || nx < 4 || ny < 4 || len(raw) != bw*bw {
		return d
	}
	if detDebug != nil {
		detDebug(P, gr.X, gr.Y, nx, ny, pres)
	}
	// somme cumulate delle caselle piene, per contare quelle dentro un rettangolo
	W := nx + 1
	cum := make([]int, (nx+1)*(ny+1))
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			v := 0
			if pres[j*nx+i] {
				v = 1
			}
			cum[(j+1)*W+i+1] = cum[j*W+i+1] + cum[(j+1)*W+i] - cum[j*W+i] + v
		}
	}
	count := func(x0, y0, x1, y1 int) int {
		x0, y0 = max(x0, 0), max(y0, 0)
		x1, y1 = min(x1, nx), min(y1, ny)
		if x1 <= x0 || y1 <= y0 {
			return 0
		}
		return cum[y1*W+x1] - cum[y0*W+x1] - cum[y1*W+x0] + cum[y0*W+x0]
	}
	bestScore, bestR, bestDX, bestDY := -1e9, 0, 0, 0
	for r := 0; r < 4; r++ {
		m := rotateGrid(raw, bw, r)
		var nodes [][2]int
		minI, maxI, minJ, maxJ := bw, -1, bw, -1
		for i := 0; i < len(m); i++ {
			if m[i] != '0' {
				x, y := i%bw, i/bw
				nodes = append(nodes, [2]int{x, y})
				minI, maxI, minJ, maxJ = min(minI, x), max(maxI, x), min(minJ, y), max(maxJ, y)
			}
		}
		total := len(nodes)
		if total == 0 {
			continue
		}
		// in ordine sparso, così le posizioni sbagliate si scartano subito
		for i := range nodes {
			j := (i*7919 + 13) % len(nodes)
			nodes[i], nodes[j] = nodes[j], nodes[i]
		}
		maxMiss := int(0.14*float64(total)) + 1
		for dy := -maxJ; dy < ny-minJ; dy++ {
			for dx := -maxI; dx < nx-minI; dx++ {
				inView, hits, miss := 0, 0, 0
				for _, n := range nodes {
					xx, yy := dx+n[0], dy+n[1]
					if xx < 0 || yy < 0 || xx >= nx || yy >= ny {
						continue
					}
					inView++
					if pres[yy*nx+xx] {
						hits++
					} else if miss++; miss > maxMiss {
						break // troppi buchi: qui non può esserci la tavola
					}
				}
				if miss > maxMiss {
					continue
				}
				if float64(inView) < 0.4*float64(total) || inView < 18 {
					continue
				}
				extra := count(dx+minI, dy+minJ, dx+maxI+1, dy+maxJ+1) - hits // pieni dove la tavola è vuota
				sc := (float64(hits) - 0.6*float64(inView-hits) - 0.4*float64(extra)) / float64(inView)
				if sc > bestScore {
					bestScore, bestR, bestDX, bestDY = sc, r, dx, dy
				}
			}
		}
	}
	if bestScore < 0.8 {
		d.Score = bestScore
		return d
	}
	return BoardDetect{OK: true, X: gr.X + float64(bestDX)*P, Y: gr.Y + float64(bestDY)*P, Cell: P, Rot: bestR, Score: bestScore}
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

// contrasto (deviazione standard) di ogni casella, con le immagini integrali
type integ struct {
	w, h  int
	s, s2 []float64
}

func newInteg(g []float64, w, h int) *integ {
	in := &integ{w: w, h: h, s: make([]float64, (w+1)*(h+1)), s2: make([]float64, (w+1)*(h+1))}
	for y := 0; y < h; y++ {
		var r, r2 float64
		for x := 0; x < w; x++ {
			v := g[y*w+x]
			r += v
			r2 += v * v
			i := (y+1)*(w+1) + x + 1
			in.s[i] = in.s[i-(w+1)] + r
			in.s2[i] = in.s2[i-(w+1)] + r2
		}
	}
	return in
}

func (in *integ) std(x0, y0, x1, y1 int) float64 {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > in.w {
		x1 = in.w
	}
	if y1 > in.h {
		y1 = in.h
	}
	n := float64((x1 - x0) * (y1 - y0))
	if n <= 0 {
		return 0
	}
	W := in.w + 1
	a := in.s[y1*W+x1] - in.s[y0*W+x1] - in.s[y1*W+x0] + in.s[y0*W+x0]
	b := in.s2[y1*W+x1] - in.s2[y0*W+x1] - in.s2[y1*W+x0] + in.s2[y0*W+x0]
	v := b/n - (a/n)*(a/n)
	if v < 0 {
		return 0
	}
	return math.Sqrt(v)
}

// separazione in due gruppi (metodo di Otsu): quanto bene si dividono, e la soglia
func otsu(v []float64) (float64, float64) {
	if len(v) < 8 {
		return 0, 0
	}
	sv := append([]float64(nil), v...)
	sort.Float64s(sv)
	n := float64(len(sv))
	var tot, tot2 float64
	for _, x := range sv {
		tot += x
		tot2 += x * x
	}
	mean := tot / n
	varT := tot2/n - mean*mean
	if varT <= 1e-9 {
		return 0, 0
	}
	best, thr := 0.0, 0.0
	var c float64
	for i := 0; i < len(sv)-1; i++ {
		c += sv[i]
		w0 := float64(i+1) / n
		m0 := c / float64(i+1)
		m1 := (tot - c) / (n - float64(i+1))
		bv := w0 * (1 - w0) * (m0 - m1) * (m0 - m1)
		if bv > best {
			best, thr = bv, (sv[i]+sv[i+1])/2
		}
	}
	return best / varT, thr
}

func gridStds(in *integ, P, ox, oy float64) ([]float64, int, int) {
	nx, ny := int((float64(in.w)-ox)/P), int((float64(in.h)-oy)/P)
	if nx < 1 || ny < 1 {
		return nil, 0, 0
	}
	out := make([]float64, nx*ny)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			cx, cy := ox+float64(i)*P, oy+float64(j)*P
			out[j*nx+i] = in.std(int(cx+P*0.22), int(cy+P*0.22), int(cx+P*0.78), int(cy+P*0.78))
		}
	}
	return out, nx, ny
}

func findGrids(g []float64, w, h, minP, maxP int) []PGrid {
	in := newInteg(g, w, h)
	// candidati per il passo: picchi dell'autocorrelazione dei bordi e i loro multipli
	cols := make([]float64, w-1)
	rows := make([]float64, h-1)
	for y := 0; y < h-1; y++ {
		o := y * w
		for x := 0; x < w-1; x++ {
			cols[x] += absf(g[o+x+1] - g[o+x])
			rows[y] += absf(g[o+w+x] - g[o+x])
		}
	}
	cand := map[int]bool{}
	for _, prof := range [][]float64{cols, rows} {
		for _, l := range periodPeaks(prof, max(8, minP/2), maxP, 6) {
			for k := 1; k <= 3 && k*l <= maxP; k++ {
				if k*l >= minP {
					cand[k*l] = true
				}
			}
		}
	}
	try := func(P, ox, oy float64) float64 {
		v, nx, ny := gridStds(in, P, ox, oy)
		if nx < 4 || ny < 4 {
			return 0
		}
		q, _ := otsu(v)
		return q
	}
	// per ogni passo la fase migliore; quale passo sia giusto lo decide il confronto con le tavole
	if len(cand) == 0 && maxP-minP <= 12 { // si segue una tavola già trovata: si prova il suo passo
		cand[(minP+maxP)/2] = true
	}
	// passi quasi uguali: basta provarne uno
	var cl []int
	for c := range cand {
		cl = append(cl, c)
	}
	sort.Ints(cl)
	var uniq []int
	for _, c := range cl {
		if len(uniq) == 0 || float64(c) > float64(uniq[len(uniq)-1])*1.04 {
			uniq = append(uniq, c)
		}
	}
	var out []PGrid
	for _, c := range uniq {
		P := float64(c)
		bq, box, boy := 0.0, 0.0, 0.0
		step := math.Max(1, P/12)
		for oy := 0.0; oy < P; oy += step {
			for ox := 0.0; ox < P; ox += step {
				if q := try(P, ox, oy); q > bq {
					bq, box, boy = q, ox, oy
				}
			}
		}
		if bq < 0.6 {
			continue
		}
		for dP := -0.9; dP <= 0.9; dP += 0.3 {
			P2 := P + dP
			for dy := -step; dy <= step; dy++ {
				for dx := -step; dx <= step; dx++ {
					ox, oy := math.Mod(box+dx+P2, P2), math.Mod(boy+dy+P2, P2)
					if q := try(P2, ox, oy); q > bq {
						bq, box, boy, P = q, ox, oy, P2
					}
				}
			}
		}
		// la fase migliore può cadere a metà casella: si provano anche gli spostamenti di mezza casella
		for _, sh := range [][2]float64{{0, 0}, {0.5, 0}, {0, 0.5}, {0.5, 0.5}} {
			ox, oy := math.Mod(box+sh[0]*P, P), math.Mod(boy+sh[1]*P, P)
			v, nx, ny := gridStds(in, P, ox, oy)
			if nx < 4 || ny < 4 {
				continue
			}
			_, thr := otsu(v)
			pres := make([]bool, len(v))
			np := 0
			for i, x := range v {
				pres[i] = x > thr
				if pres[i] {
					np++
				}
			}
			if np < 18 || np > len(v)*7/10 {
				continue
			}
			out = append(out, PGrid{P: P, X: ox, Y: oy, NX: nx, NY: ny, Pres: pres})
		}
	}
	return out
}

// i picchi più alti dell'autocorrelazione di un profilo
func periodPeaks(sig []float64, minL, maxL, k int) []int {
	n := len(sig)
	if maxL > n/2 {
		maxL = n / 2
	}
	if maxL <= minL+1 {
		return nil
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
	if e0 == 0 {
		return nil
	}
	rs := make([]float64, maxL+2)
	for l := minL; l <= maxL+1 && l < n; l++ {
		var s float64
		for i := 0; i+l < n; i++ {
			s += x[i] * x[i+l]
		}
		rs[l] = s / e0 * float64(n) / float64(n-l)
	}
	type pk struct {
		l int
		r float64
	}
	var ps []pk
	for l := minL + 1; l <= maxL; l++ {
		if rs[l] > 0 && rs[l] >= rs[l-1] && rs[l] >= rs[l+1] {
			ps = append(ps, pk{l, rs[l]})
		}
	}
	sort.Slice(ps, func(a, b int) bool { return ps[a].r > ps[b].r })
	var out []int
	for i := 0; i < len(ps) && i < k; i++ {
		out = append(out, ps[i].l)
	}
	return out
}

// rifinitura della posizione: il centro di ogni nodo è il baricentro dei bordi (anello e icona)
// nella sua casella; si ricava la retta migliore per colonne e righe (posizione e passo)
func refineDetect(g []float64, w, h int, d BoardDetect, raw string, bw int) BoardDetect {
	if !d.OK {
		return d
	}
	m := rotateGrid(raw, bw, d.Rot)
	for it := 0; it < 3; it++ {
		var sx, sxx, sxy, sy, n float64 // regressione: centro x = X + (i+0.5)*P
		var ty, tyy, tyc, tc, tn float64
		P := d.Cell
		for idx := 0; idx < len(m); idx++ {
			if m[idx] == '0' {
				continue
			}
			i, j := idx%bw, idx/bw
			x0, y0 := int(d.X+float64(i)*P), int(d.Y+float64(j)*P)
			x1, y1 := int(d.X+float64(i+1)*P), int(d.Y+float64(j+1)*P)
			if x0 < 1 || y0 < 1 || x1 >= w-1 || y1 >= h-1 {
				continue
			}
			var ex, ey, es float64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					e := absf(g[y*w+x+1]-g[y*w+x-1]) + absf(g[(y+1)*w+x]-g[(y-1)*w+x])
					ex += e * float64(x)
					ey += e * float64(y)
					es += e
				}
			}
			if es <= 0 {
				continue
			}
			cx, cy := ex/es, ey/es
			fi, fj := float64(i)+0.5, float64(j)+0.5
			sx += fi
			sxx += fi * fi
			sxy += fi * cx
			sy += cx
			n++
			ty += fj
			tyy += fj * fj
			tyc += fj * cy
			tc += cy
			tn++
		}
		if n < 10 {
			return d
		}
		den := n*sxx - sx*sx
		den2 := tn*tyy - ty*ty
		if den <= 0 || den2 <= 0 {
			return d
		}
		px := (n*sxy - sx*sy) / den
		py := (tn*tyc - ty*tc) / den2
		np := (px + py) / 2
		if absf(np-d.Cell) > 0.08*d.Cell {
			np = d.Cell
		}
		// posizione con il passo comune
		var ax, ay float64
		ax = (sy - np*sx) / n
		ay = (tc - np*ty) / tn
		d.X, d.Y, d.Cell = ax, ay, np
	}
	return d
}

func halfGray(g []float64, w, h int) ([]float64, int, int) {
	w2, h2 := w/2, h/2
	out := make([]float64, w2*h2)
	for y := 0; y < h2; y++ {
		for x := 0; x < w2; x++ {
			i := 2*y*w + 2*x
			out[y*w2+x] = (g[i] + g[i+1] + g[i+w] + g[i+w+1]) / 4
		}
	}
	return out, w2, h2
}
