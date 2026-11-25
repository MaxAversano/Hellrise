package main

// Build di D4Builds: la pagina pubblica, letta come la vede un visitatore (con Edge nascosto),
// da cui si prendono equipaggiamento, abilità e rune, poi tradotti con i dati di gioco.

import (
	"crypto/sha1"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var d4bMu sync.Mutex

var slotIt = map[string]string{"Helm": "Elmo", "Chest Armor": "Torso", "Gloves": "Guanti", "Pants": "Pantaloni", "Boots": "Stivali",
	"Amulet": "Amuleto", "Ring 1": "Anello 1", "Ring 2": "Anello 2", "Weapon": "Arma", "Offhand": "Mano secondaria",
	"Bludgeoning Weapon": "Arma contundente", "Slashing Weapon": "Arma da taglio", "Dual-Wield Weapon 1": "Arma (mano 1)",
	"Dual-Wield Weapon 2": "Arma (mano 2)", "Ranged Weapon": "Arma a distanza", "Focus": "Focus", "Totem": "Totem", "Shield": "Scudo"}

//go:embed d4b_extract.js
var d4bExtractJS string

// Build D4Builds completa, com'è sul sito: tutte le schede (equipaggiamento e abilità, albero, paragon,
// mercenari, piani di guerra, note) lette nel lettore nascosto, più i nomi italiani in "tr".
type D4Full struct {
	OK       int               `json:"ok"`
	Err      string            `json:"err,omitempty"`
	URL      string            `json:"url"`
	At       int64             `json:"at"`
	Head     map[string]string `json:"head"`
	Variants []map[string]any  `json:"variants"`
	Notes    any               `json:"notes"`
	NotesT   string            `json:"notesTitle"`
	Tr       map[string]string `json:"tr"`
	XV       int               `json:"xv"`
	Filters  []PFilter         `json:"filters,omitempty"` // filtri loot (Maxroll)
	Planner  string            `json:"planner,omitempty"` // id del planner Maxroll
	Src      string            `json:"src,omitempty"`
}

const d4bExtractVer = 3 // da aumentare quando l'estrattore legge cose nuove

func d4bFullRead(u string) (*D4Full, error) {
	if !strings.HasPrefix(u, "https://d4builds.gg/builds/") {
		return nil, &httpErr{400}
	}
	p := filepath.Join(dataDir, "cache", "d4bfull_"+hashURL(u)+".json")
	var f D4Full
	if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 6*time.Hour && readJSON(p, &f) == nil && f.OK == 1 && f.XV == d4bExtractVer {
		return &f, nil
	}
	d4bMu.Lock()
	got, err := browserRun(u, `!!document.querySelector('.builder__gear__name')`, d4bExtractJS, `window.__hrOut||null`, 4*time.Minute)
	d4bMu.Unlock()
	if err == nil {
		err = json.Unmarshal([]byte(got), &f)
	}
	if err == nil && f.OK != 1 {
		err = fmt.Errorf("lettura D4Builds: %s", f.Err)
	}
	if err != nil {
		var old D4Full
		if readJSON(p, &old) == nil && old.OK == 1 {
			return &old, nil
		}
		return nil, err
	}
	f.URL, f.At, f.XV = u, time.Now().Unix(), d4bExtractVer
	f.Tr = d4bTranslate(&f)
	writeJSON(p, &f)
	return &f, nil
}

// nomi inglesi → italiano (oggetti, aspetti, abilità, glifi, tavole, nodi)
var trCache struct {
	sync.Mutex
	d *gameData
	m map[string]string
}

func trMap() map[string]string {
	d := getGD()
	trCache.Lock()
	defer trCache.Unlock()
	if d == nil {
		return map[string]string{}
	}
	if trCache.d == d {
		return trCache.m
	}
	m := map[string]string{}
	add := func(it, en string) {
		if it != "" && en != "" && it != en {
			if _, ok := m[normName(en)]; !ok {
				m[normName(en)] = it
			}
		}
	}
	for _, g := range d.Items {
		add(g.It, g.En)
	}
	for _, a := range d.Aspects {
		add(a[0], a[1])
	}
	for _, sk := range d.Skills {
		add(sk[0], sk[1])
	}
	for _, g := range d.Glyphs {
		add(g[0], g[1])
	}
	for _, b := range d.Boards {
		add(b.It, b.En)
	}
	for _, n := range d.NodeNames {
		add(n[0], n[1])
	}
	for en, it := range slotIt {
		add(it, en)
	}
	trCache.d, trCache.m = d, m
	return m
}

func trName(en string) string {
	m := trMap()
	if it := m[normName(en)]; it != "" {
		return it
	}
	if it, g := itName(en); g != nil {
		return it
	}
	if x := regexp.MustCompile(`\s*\(.*\)$`).ReplaceAllString(en, ""); x != en {
		if it, g := itName(x); g != nil {
			return it
		}
	}
	// statistiche degli oggetti (es. "Critical Strike Damage Multiplier")
	if g := gxLoad(); g != nil {
		if a, ok := g.statSlug[textSlug(en)]; ok && a.StatIt != "" {
			return a.StatIt
		}
	}
	return ""
}

func d4bTranslate(f *D4Full) map[string]string {
	out := map[string]string{}
	var walk func(v any, key string)
	walk = func(v any, key string) {
		switch x := v.(type) {
		case map[string]any:
			for k, vv := range x {
				walk(vv, k)
			}
		case []any:
			for _, vv := range x {
				walk(vv, key)
			}
		case string:
			if (key == "n" || key == "slot" || key == "g" || key == "t") && len(x) > 1 && len(x) < 80 {
				if _, ok := out[x]; !ok {
					if it := trName(x); it != "" {
						out[x] = it
					}
					if _, gi := itName(x); gi != nil && gi.Mythic {
						out["~m:"+x] = "1"
					}
				}
			}
		}
	}
	for _, v := range f.Variants {
		walk(v, "")
		if t, ok := v["tree"].(map[string]any); ok {
			if nodes, ok := t["nodes"].([]any); ok {
				for _, n := range nodes {
					if a, ok := n.([]any); ok && len(a) > 3 {
						if s, ok := a[3].(string); ok && s != "" {
							if it := trName(s); it != "" {
								out[s] = it
							}
						}
					}
				}
			}
		}
	}
	walk(f.Notes, "")
	return out
}

// versione "leggera" (equipaggiamento, abilità, rune) usata per elenchi, ricerca e overlay
func (f *D4Full) slim() *PBuild {
	str := func(m map[string]any, k string) string { s, _ := m[k].(string); return s }
	list := func(m map[string]any, k string) []map[string]any {
		var out []map[string]any
		if a, ok := m[k].([]any); ok {
			for _, x := range a {
				if mm, ok := x.(map[string]any); ok {
					out = append(out, mm)
				}
			}
		}
		return out
	}
	tr := func(en string) string {
		if it := f.Tr[en]; it != "" {
			return it
		}
		return en
	}
	b := &PBuild{Name: f.Head["title"], URL: f.URL}
	if t := d4bDate(f.Head["updated"]); t > 0 {
		b.Updated = strconv.FormatInt(t, 10)
	}
	for _, v := range f.Variants {
		pv := PVariant{Name: str(v, "name")}
		for _, g := range list(v, "gear") {
			en, slot := str(g, "n"), str(g, "slot")
			if en == "" || strings.EqualFold(slot, "Empty") {
				continue
			}
			pg := PGear{En: en, Name: tr(en), TypeEn: slot, Type: tr(slot), Kind: "unique"}
			if k := str(g, "k"); k == "mythic" {
				pg.Kind = "mythic"
			}
			if strings.Contains(strings.ToLower(en), "aspect") {
				pg.Kind = "legendary"
			}
			for _, gm := range list(g, "gems") {
				if n := str(gm, "n"); n != "" {
					pg.Sockets = append(pg.Sockets, tr(n)+"|"+n)
				}
			}
			pv.Gear = append(pv.Gear, pg)
		}
		var runes []string
		for _, r := range list(v, "runes") {
			if n := str(r, "n"); n != "" {
				runes = append(runes, tr(n)+"|"+n)
			}
		}
		if len(runes) > 0 {
			pv.Gear = append(pv.Gear, PGear{Type: "Rune attive", TypeEn: "Active Runes", Name: "Rune", En: "Runes", Kind: "other", Sockets: runes})
		}
		for _, sk := range list(v, "skills") {
			if n := str(sk, "n"); n != "" && len(pv.Skills) < 6 {
				pv.Skills = append(pv.Skills, PSkill{Key: squash(n), Name: tr(n), En: n})
			}
		}
		b.Variants = append(b.Variants, pv)
	}
	return b
}

// "Updated on November 12, 2026." → data unix
func d4bDate(s string) int64 {
	s = strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "Updated on")), ".")
	if t, err := time.Parse("January 2, 2006", strings.TrimSpace(s)); err == nil {
		return t.Unix()
	}
	return 0
}

func d4bBuildDetail(u string) (*PBuild, error) {
	f, err := d4bFullRead(u)
	if err != nil {
		return nil, err
	}
	b := f.slim()
	if len(b.Variants) == 0 {
		return nil, &httpErr{422}
	}
	return b, nil
}

func squash(s string) string {
	return strings.ToLower(strings.NewReplacer(" ", "", "'", "", "’", "", "-", "", ":", "").Replace(s))
}

func hashURL(u string) string {
	sum := sha1.Sum([]byte(u))
	return hex.EncodeToString(sum[:8])
}
