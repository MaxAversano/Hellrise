package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// dettagli di glifi e nodi leggendari del paragon, presi dai dati di gioco (italiano e inglese)
type GlyphBonus struct {
	It   string `json:"it"`
	En   string `json:"en"`
	Rank int    `json:"rank"` // 0 sempre, 1 dal livello 15, 2 dal livello 46
}

type GlyphInfo struct {
	It    string       `json:"it"`
	En    string       `json:"en"`
	ReqIt string       `json:"reqIt,omitempty"` // "40 Intelligenza a portata"
	ReqEn string       `json:"reqEn,omitempty"`
	Bonus []GlyphBonus `json:"bonus"`
}

type NodeInfo struct {
	It     string `json:"it"`
	En     string `json:"en"`
	DescIt string `json:"descIt"`
	DescEn string `json:"descEn"`
}

type ParagonInfo struct {
	Glyphs map[string]GlyphInfo `json:"glyphs"` // chiave: nome inglese in minuscolo
	Nodes  map[string]NodeInfo  `json:"nodes"`  // chiave: id del nodo (rari e leggendari)
	Board  map[string]string    `json:"board"`  // nome inglese della tavola in minuscolo → nodo leggendario
}

var pinfoCache struct {
	sync.Mutex
	d   *gameData
	val *ParagonInfo
}

var reScalar = regexp.MustCompile(`\[\{GlyphAffixScalar\}\|([^|\]]*)\|\]`)

var reDec = regexp.MustCompile(`(\d)\.(\d)`)

// numeri all'italiana e via i "[+]" del testo di gioco
func itNum(s string) string {
	return strings.ReplaceAll(reDec.ReplaceAllString(s, "$1,$2"), "[+]", "")
}

func num(f float64) string {
	s := strconv.FormatFloat(f, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func paragonInfo() *ParagonInfo {
	d := getGD()
	pinfoCache.Lock()
	defer pinfoCache.Unlock()
	if pinfoCache.val != nil && pinfoCache.d == d {
		return pinfoCache.val
	}
	type aff struct {
		Op      int      `json:"operation"`
		Rank    int      `json:"requiredRank"`
		Base    float64  `json:"base"`
		Per     float64  `json:"perLevel"`
		Thresh  []string `json:"thresholds"`
		Power   string   `json:"power"`
		Formula string   `json:"formula"`
	}
	var mn struct {
		Glyphs map[string]struct {
			Affixes []string `json:"affixes"`
		} `json:"paragonGlyphs"`
		Affixes map[string]aff `json:"paragonGlyphAffixes"`
		Thresh  map[string]struct {
			Attributes []struct {
				ID    int `json:"id"`
				Value any `json:"value"`
			} `json:"attributes"`
		} `json:"paragonThresholds"`
		Nodes map[string]struct {
			Rarity int    `json:"rarity"`
			Power  string `json:"power"`
		} `json:"paragonNodes"`
		Boards map[string]struct {
			Nodes []*string `json:"nodes"`
		} `json:"paragonBoards"`
	}
	type nd struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	type lang struct {
		Glyphs  map[string]nd `json:"paragonGlyphs"`
		Affixes map[string]nd `json:"paragonGlyphAffixes"`
		Nodes   map[string]nd `json:"paragonNodes"`
		Skills  map[string]nd `json:"skills"`
	}
	var it, en lang
	if readJSON(cacheFile("data.min.json"), &mn) != nil || readJSON(cacheFile("data.itit.json"), &it) != nil || readJSON(cacheFile("data.enus.json"), &en) != nil {
		return &ParagonInfo{}
	}
	stat := map[int][2]string{9: {"Forza", "Strength"}, 10: {"Intelligenza", "Intelligence"}, 11: {"Volontà", "Willpower"}, 12: {"Destrezza", "Dexterity"}}
	// valore del bonus dal livello 1 al 100, con la stessa scala del gioco
	scal := func(a aff) string {
		v1, v100 := a.Base, a.Base+a.Per*99
		switch a.Op {
		case 1:
			v1, v100 = v1/10, v100/10
		case 4:
			v1, v100 = v1*100, v100*100
		}
		return num(v1) + "→" + num(v100)
	}
	fill := func(tpl string, a aff) string {
		tpl = reScalar.ReplaceAllStringFunc(tpl, func(m string) string {
			suf := reScalar.FindStringSubmatch(m)[1]
			unit := ""
			if strings.Contains(suf, "%") {
				unit = "%"
			}
			if strings.Contains(suf, "x") {
				unit = "%x"
			}
			return scal(a) + unit
		})
		return gameText(tpl)
	}
	out := &ParagonInfo{Glyphs: map[string]GlyphInfo{}, Nodes: map[string]NodeInfo{}, Board: map[string]string{}}
	for k, g := range mn.Glyphs {
		gi := GlyphInfo{It: cleanName(it.Glyphs[k].Name), En: cleanName(en.Glyphs[k].Name)}
		if gi.En == "" {
			continue
		}
		for _, ak := range g.Affixes {
			a := mn.Affixes[ak]
			if a.Op == 5 {
				for _, t := range a.Thresh {
					for _, at := range mn.Thresh[t].Attributes {
						if s, ok := stat[at.ID]; ok {
							v := ""
							switch x := at.Value.(type) {
							case float64:
								v = num(x)
							case string:
								v = x
							}
							gi.ReqIt, gi.ReqEn = v+" "+s[0], v+" "+s[1]
						}
					}
				}
				if a.Power != "" {
					gi.Bonus = append(gi.Bonus, GlyphBonus{It: itNum(gameText(it.Skills[a.Power].Desc)), En: gameText(en.Skills[a.Power].Desc), Rank: -1})
				}
				continue
			}
			bi, be := itNum(fill(it.Affixes[ak].Desc, a)), fill(en.Affixes[ak].Desc, a)
			if bi == "" && be == "" {
				continue
			}
			gi.Bonus = append(gi.Bonus, GlyphBonus{It: bi, En: be, Rank: a.Rank})
		}
		out.Glyphs[strings.ToLower(gi.En)] = gi
	}
	for k, n := range mn.Nodes {
		if n.Rarity < 3 {
			continue
		}
		ni := NodeInfo{It: cleanName(it.Nodes[k].Name), En: cleanName(en.Nodes[k].Name)}
		if n.Power != "" {
			ni.DescIt, ni.DescEn = itNum(gameText(it.Skills[n.Power].Desc)), gameText(en.Skills[n.Power].Desc)
		}
		if ni.En != "" {
			out.Nodes[k] = ni
		}
	}
	for k, b := range mn.Boards {
		en := strings.ToLower(getGD().Boards[k].En)
		for _, x := range b.Nodes {
			if x != nil && mn.Nodes[*x].Rarity >= 4 && en != "" {
				out.Board[en] = *x
			}
		}
	}
	pinfoCache.d, pinfoCache.val = d, out
	return out
}
