package main

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

func TestBuildView(t *testing.T) {
	raw := `{"season":16,"maxroll":[{"slug":"whirlwind-barbarian-guide","title":"Whirlwind Barbarian Endgame Build Guide","cls":"barbarian","type":"Endgame","profile":"qeyb7t0x","url":"https://maxroll.gg/d4/build-guides/whirlwind-barbarian-guide"}],
	"d4builds":[{"slug":"flay-barbarian-endgame","name":"Flay Barbarian","cls":"barbarian","type":"Endgame","url":"https://d4builds.gg/builds/flay-barbarian-endgame/"}],
	"rob":{"tier":[{"tier":"S","build":"Flay Barb","info":"150","url":"https://d4builds.gg/builds/flay-barbarian-endgame","cls":"barbarian","rank":1},{"tier":"A","build":"Whirlwind Barb","info":"147","url":"https://d4builds.gg/builds/1ea6e10e-05a9-4231-b178-c0b1cb98ad37/?var=1","cls":"barbarian","rank":2}],"leveling":[{"tier":"S","build":"Whirlwind Barb","url":"https://d4builds.gg/builds/whirlwind-barbarian-leveling","cls":"barbarian"}]},
	"loot":{"source":"d4builds.gg","bosses":[{"boss":"All Bosses","name":"All Bosses","all":["Azurewrath"]},{"boss":"Varshan","name":"Echo of Varshan","location":"Malignant Burrow","all":["Shard of Verathiel"],"byClass":{"barbarian":["Overkill"]}}]},
	"lootMaxroll":{"sections":[{"name":"Echo of Varshan","items":[{"id":1210594,"name":"100,000 Steps"}]},{"name":"Mythics","items":[{"id":241930,"name":"Andariel's Visage"}]}]},
	"bossInfo":{"bosses":[{"name":"Echo of Varshan","lines":["Material Required: 1 Lair Key."]}]}}`
	var s seasonRaw
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	v := buildView(&s)
	for _, b := range v.Builds {
		t.Logf("%s | %s | tier=%s", b.Src, b.Title, b.Tier)
	}
	if v.Builds[0].Tier != "A" || v.Builds[1].Tier != "S" {
		t.Fatalf("tier non trovati")
	}
	for _, b := range v.Bosses {
		t.Logf("boss %s (%s) items=%v info=%v", b.Name, b.ID, b.Items, b.Info)
	}
	for _, it := range v.Items {
		t.Logf("item %s %+v", it.Name, it.Sources)
	}
}

func TestEvents(t *testing.T) {
	for _, e := range upcomingEvents(time.Now(), 3) {
		t.Logf("%s %d-%d %s", e.Kind, e.Start, e.End, e.Title)
	}
}

func TestPeriod(t *testing.T) {
	// griglia di caselle quadrate da 37 px (bordo chiaro), con rumore
	w, h := 700, 500
	g := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := 20.0
			cx, cy := (x+5)%37, (y+11)%37
			if cx >= 4 && cx <= 31 && cy >= 4 && cy <= 31 && (x/37+y/37)%3 != 0 {
				v = 120
			}
			v += float64((x*7919+y*104729)%23) - 11
			g[y*w+x] = v
		}
	}
	cols := make([]float64, w-1)
	for y := 0; y < h; y++ {
		for x := 0; x < w-1; x++ {
			cols[x] += absf(g[y*w+x+1] - g[y*w+x])
		}
	}
	p, r := dominantPeriod(cols, 14, 130)
	t.Logf("periodo %d forza %.2f", p, r)
	if p != 37 {
		t.Fatalf("atteso 37, trovato %d", p)
	}
	if rotateGrid("abcd", 2, 1) != "cadb" {
		t.Fatalf("rotazione: %s", rotateGrid("abcd", 2, 1))
	}
}

func TestDetectBoard(t *testing.T) {
	// tavola finta 21x21 con forma irregolare, ruotata di 1 quarto e disegnata a passo 33 px
	bw := 21
	raw := make([]byte, bw*bw)
	for i := range raw {
		x, y := i%bw, i/bw
		raw[i] = '0'
		if (x == 10) || (y == 10 && x > 3) || (x+y)%5 == 0 && x > 2 && y < 15 || (x == 3 && y > 12) {
			raw[i] = '1'
		}
	}
	shown := rotateGrid(string(raw), bw, 1)
	P := 33
	w, h := 1100, 900
	g := make([]float64, w*h)
	ox, oy := 77, 61
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g[y*w+x] = 18 + float64((x*31+y*17)%7)
		}
	}
	for j := 0; j < bw; j++ {
		for i := 0; i < bw; i++ {
			if shown[j*bw+i] == '0' {
				continue
			}
			// nodo come nel gioco: anello chiaro e icona al centro
			cx, cy := float64(ox+i*P)+float64(P)/2, float64(oy+j*P)+float64(P)/2
			for y := oy + j*P; y < oy+(j+1)*P; y++ {
				for x := ox + i*P; x < ox+(i+1)*P; x++ {
					r := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
					switch {
					case r > 0.36*float64(P) && r < 0.44*float64(P):
						g[y*w+x] = 170
					case r < 0.22*float64(P):
						if (x+y)%4 < 2 {
							g[y*w+x] = 200
						} else {
							g[y*w+x] = 60
						}
					case r < 0.36*float64(P):
						g[y*w+x] = 45
					}
				}
			}
		}
	}
	d := detectBoard(g, w, h, string(raw), bw, 14, 130)
	t.Logf("%+v", d)
	if !d.OK || d.Rot != 1 || absf(d.X-float64(ox)) > 3 || absf(d.Y-float64(oy)) > 3 || absf(d.Cell-float64(P)) > 1 {
		t.Fatalf("tavola non riconosciuta bene")
	}
}

func TestRealSeason(t *testing.T) {
	b, err := os.ReadFile("testdata/s15.json")
	if err != nil {
		t.Skip()
	}
	var s seasonRaw
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	v := buildView(&s)
	tiers := 0
	for _, x := range v.Builds {
		if x.Tier != "" {
			tiers++
		}
	}
	t.Logf("build %d (con tier %d), oggetti %d, boss %d", len(v.Builds), tiers, len(v.Items), len(v.Bosses))
	if len(v.Builds) < 50 || len(v.Items) < 50 {
		t.Fatal("troppo pochi dati")
	}
}
