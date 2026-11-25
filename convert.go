package main

// Conversione delle build Maxroll e Mobalytics nello stesso formato completo di D4Builds
// (equipaggiamento con dettagli, talismani e set, albero delle abilità, paragon, mercenari,
// piani di guerra, note), con i nomi italiani presi dai dati di gioco.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type gxAffix struct{ Key, StatIt, StatEn, DescIt, DescEn string }
type gxNode struct {
	ID     int                    `json:"id"`
	Pos    struct{ X, Y float64 } `json:"pos"`
	Reward string                 `json:"rewardId"`
	Root   int                    `json:"root"`
}
type gxTree struct {
	Nodes []gxNode `json:"nodes"`
	Conns []struct {
		A int `json:"node1"`
		B int `json:"node2"`
	} `json:"connections"`
}
type gxReward struct {
	Type  int    `json:"type"`
	Power string `json:"power"`
	Ranks int    `json:"ranks"`
	Mod   uint32 `json:"mod"`
}
type gxSkill struct {
	It, En string
	Mods   map[uint32][2]string
}
type gxItem struct {
	Type, Set string
	Magic     int
	Explicits []string
}
type gxSet struct {
	It, En  string
	Bonuses []struct {
		Req   int    `json:"required"`
		Affix string `json:"affix"`
	}
	Pieces []string
}
type gxData struct {
	affix    map[uint32]gxAffix
	affixKey map[string]gxAffix
	trees    map[string]gxTree
	rewards  map[string]gxReward
	skills   map[string]gxSkill
	items    map[string]gxItem
	sets     map[string]gxSet
	mercs    map[string][2]string
	war      []struct{ Tree, It, En string }
	statSlug map[string]gxAffix // nome inglese della statistica in forma "slug" → affisso
}

var gxCache struct {
	sync.Mutex
	d   *gameData
	val *gxData
}

var (
	reTag    = regexp.MustCompile(`\{/?c[^}]*\}|\{/?if[^}]*\}|\{else\}|\{/?u\}|\{icon:[^}]*\}`)
	reVal    = regexp.MustCompile(`\[([^\]\|]*)\|[^\]]*\]`)
	reBrace  = regexp.MustCompile(`\{[^}]*\}`)
	reSpaces = regexp.MustCompile(`\s+`)
)

// testo di gioco leggibile: via i tag di colore/condizioni, valori calcolati come "#"
func gameText(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = reTag.ReplaceAllString(s, "")
	s = reVal.ReplaceAllStringFunc(s, func(m string) string {
		in := reVal.FindStringSubmatch(m)[1]
		if regexp.MustCompile(`^[\d\s.,%x+\-–]+$`).MatchString(in) {
			return in
		}
		pct := strings.Contains(m, "%")
		if pct {
			return "#%"
		}
		return "#"
	})
	s = reBrace.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, `\[x\]`, "x")
	s = strings.ReplaceAll(s, `\`, "")
	s = strings.ReplaceAll(s, "[x]", "x")
	s = strings.ReplaceAll(s, "[ms]", "")
	s = strings.ReplaceAll(s, "[fs]", "")
	s = strings.ReplaceAll(s, "[+]", "")
	return strings.TrimSpace(reSpaces.ReplaceAllString(s, " "))
}

// nome della statistica senza valori: "Forza: [{value}|~|]" → "Forza"
func statName(tpl string) string {
	s := reVal.ReplaceAllString(tpl, "")
	s = reBrace.ReplaceAllString(reTag.ReplaceAllString(s, ""), "")
	s = strings.Trim(strings.TrimSpace(reSpaces.ReplaceAllString(s, " ")), ":+x× ")
	s = strings.TrimSuffix(strings.TrimSpace(s), ":")
	return strings.TrimSpace(strings.TrimPrefix(s, "x "))
}

func textSlug(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`['’]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func gxLoad() *gxData {
	d := getGD()
	gxCache.Lock()
	defer gxCache.Unlock()
	if gxCache.val != nil && gxCache.d == d {
		return gxCache.val
	}
	type nameDesc struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
		Mods []struct {
			Name string `json:"name"`
		} `json:"mods"`
	}
	var mn struct {
		Affixes map[string]struct {
			ID         uint32 `json:"id"`
			Attributes []struct {
				ID    int  `json:"id"`
				Param *int `json:"param"`
			} `json:"attributes"`
		} `json:"affixes"`
		Attributes map[string]struct {
			Name string `json:"name"`
		} `json:"attributes"`
		Skills map[string]struct {
			Mods []struct {
				ID uint32 `json:"id"`
			} `json:"mods"`
		} `json:"skills"`
		Trees   map[string]gxTree   `json:"skillTrees"`
		Rewards map[string]gxReward `json:"skillTreeRewards"`
		Items   map[string]struct {
			Type      string   `json:"type"`
			Set       string   `json:"set"`
			Magic     int      `json:"magicType"`
			Explicits []string `json:"explicits"`
		} `json:"items"`
		ItemSets map[string]struct {
			Bonuses []struct {
				Req   int    `json:"required"`
				Affix string `json:"affix"`
			} `json:"bonuses"`
		} `json:"itemSets"`
		War []struct {
			Tree string `json:"tree"`
		} `json:"warPlans"`
	}
	type lang struct {
		Affixes  map[string]nameDesc `json:"affixes"`
		AttrDesc map[string]string   `json:"attributeDescriptions"`
		Skills   map[string]nameDesc `json:"skills"`
		ItemSets map[string]nameDesc `json:"itemSets"`
		Mercs    map[string]nameDesc `json:"mercenaries"`
		War      []nameDesc          `json:"warPlans"`
	}
	var it, en lang
	if readJSON(cacheFile("data.min.json"), &mn) != nil || readJSON(cacheFile("data.itit.json"), &it) != nil || readJSON(cacheFile("data.enus.json"), &en) != nil {
		return nil
	}
	g := &gxData{affix: map[uint32]gxAffix{}, affixKey: map[string]gxAffix{}, trees: mn.Trees, rewards: mn.Rewards, skills: map[string]gxSkill{}, items: map[string]gxItem{}, sets: map[string]gxSet{}, mercs: map[string][2]string{}, statSlug: map[string]gxAffix{}}
	for k, a := range mn.Affixes {
		x := gxAffix{Key: k, DescIt: gameText(it.Affixes[k].Desc), DescEn: gameText(en.Affixes[k].Desc)}
		if len(a.Attributes) > 0 {
			at := a.Attributes[0]
			name := mn.Attributes[itoa(at.ID)].Name
			key := name
			if at.Param != nil {
				key = name + "#" + itoa(*at.Param)
			}
			ti, te := it.AttrDesc[key], en.AttrDesc[key]
			if ti == "" {
				ti, te = it.AttrDesc[name], en.AttrDesc[name]
			}
			x.StatIt, x.StatEn = statName(ti), statName(te)
		}
		g.affix[a.ID] = x
		g.affixKey[k] = x
		if x.StatEn != "" && !strings.Contains(x.StatEn, "#") {
			s := textSlug(x.StatEn)
			if old, ok := g.statSlug[s]; !ok || strings.HasPrefix(k, "S04_") && !strings.HasPrefix(old.Key, "S04_") {
				g.statSlug[s] = x
			}
		}
	}
	for k, s := range mn.Skills {
		x := gxSkill{It: gameText(it.Skills[k].Name), En: gameText(en.Skills[k].Name), Mods: map[uint32][2]string{}}
		for i, m := range s.Mods {
			var mi, me string
			if i < len(it.Skills[k].Mods) {
				mi = it.Skills[k].Mods[i].Name
			}
			if i < len(en.Skills[k].Mods) {
				me = en.Skills[k].Mods[i].Name
			}
			x.Mods[m.ID] = [2]string{mi, me}
		}
		g.skills[k] = x
	}
	for k, x := range mn.Items {
		g.items[k] = gxItem{Type: x.Type, Set: x.Set, Magic: x.Magic, Explicits: x.Explicits}
	}
	for k, s := range mn.ItemSets {
		g.sets[k] = gxSet{It: it.ItemSets[k].Name, En: en.ItemSets[k].Name, Bonuses: s.Bonuses}
	}
	keys := make([]string, 0, len(mn.Items))
	for k := range mn.Items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s := mn.Items[k].Set; s != "" {
			x := g.sets[s]
			x.Pieces = append(x.Pieces, k)
			g.sets[s] = x
		}
	}
	for k, m := range en.Mercs {
		g.mercs[k] = [2]string{strings.TrimSpace(strings.ReplaceAll(it.Mercs[k].Name, "[ms]", "")), strings.TrimSpace(m.Name)}
	}
	for i, w := range mn.War {
		x := struct{ Tree, It, En string }{Tree: w.Tree}
		if i < len(it.War) {
			x.It = it.War[i].Name
		}
		if i < len(en.War) {
			x.En = en.War[i].Name
		}
		g.war = append(g.war, x)
	}
	gxCache.d, gxCache.val = d, g
	return g
}

// --- pezzi comuni ---
const sa = "https://sunderarmor.com/DIABLO4/"

func snakeName(s string) string {
	s = strings.ToLower(regexp.MustCompile(`\s*\(.*?\)\s*`).ReplaceAllString(s, ""))
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "-", " ")), "_")
}

type fullBuilder struct {
	tr map[string]string
}

func (fb *fullBuilder) name(en, it string) string {
	if en != "" && it != "" && it != en {
		fb.tr[en] = it
	}
	return en
}

// albero (abilità, mercenari, piani di guerra) nel formato del visore: nodi [x,y,tipo,nome,icona,sfondo,rango,preso] e linee
func (fb *fullBuilder) treeView(g *gxData, treeName string, ranks map[int]int, scale float64) map[string]any {
	t, ok := g.trees[treeName]
	if !ok || len(t.Nodes) == 0 {
		return nil
	}
	nodes := make([]any, 0, len(t.Nodes))
	got := make([]bool, len(t.Nodes))
	for i, n := range t.Nodes {
		r := g.rewards[n.Reward]
		kind, nameEn, nameIt, icon, bg := "s", "", "", "", ""
		sk := g.skills[r.Power]
		switch {
		case n.Root == 1 || n.Reward == "":
			kind, bg = "c", sa+"SkillTree/category_active.png"
		case r.Type == 0:
			kind = "l"
			nameEn, nameIt = sk.En, sk.It
			icon = sa + "Skills/VoH2/" + snakeName(sk.En) + ".png"
		case r.Type == 1:
			if m, ok := sk.Mods[r.Mod]; ok {
				nameEn, nameIt = sk.En+" · "+m[1], sk.It+" · "+m[0]
			} else {
				nameEn, nameIt = sk.En, sk.It
			}
		default:
			nameEn, nameIt = sk.En, sk.It
			if r.Ranks == 1 {
				kind = "d"
			}
		}
		fb.name(nameEn, nameIt)
		have := ranks[n.ID]
		rank := ""
		if kind != "c" {
			max := r.Ranks
			if max == 0 {
				max = 1
			}
			if have > max {
				have = max
			}
			rank = fmt.Sprintf("%d/%d", have, max)
		}
		got[i] = kind == "c" || have > 0
		if bg == "" {
			switch kind {
			case "l":
				bg = sa + "SkillTree/skill_active.png"
			default:
				bg = sa + "SkillTree/passive_active.png"
			}
		}
		on := 0
		if got[i] {
			on = 1
		}
		nodes = append(nodes, []any{int(n.Pos.X * scale), int(n.Pos.Y * scale), kind, nameEn, icon, bg, rank, on})
	}
	var lines []any
	for _, c := range t.Conns {
		if c.A < 0 || c.B < 0 || c.A >= len(t.Nodes) || c.B >= len(t.Nodes) {
			continue
		}
		a, b := t.Nodes[c.A], t.Nodes[c.B]
		on := 0
		if got[c.A] && got[c.B] {
			on = 1
		}
		lines = append(lines, []any{int(a.Pos.X * scale), int(a.Pos.Y * scale), int(b.Pos.X * scale), int(b.Pos.Y * scale), on})
	}
	return map[string]any{"nodes": nodes, "lines": lines}
}

// talismano o sigillo con statistiche, effetti e set
func (fb *fullBuilder) charm(g *gxData, key, en, it string, stats []string, statsIt []string, present map[string]bool) map[string]any {
	gi := g.items[key]
	tip := map[string]any{"n": en}
	if gi.Magic == 2 {
		tip["my"] = 1
	}
	var vals []any
	for i, s := range stats {
		v := map[string]any{"t": s}
		if i < len(statsIt) {
			fb.name(s, statsIt[i])
		}
		vals = append(vals, v)
	}
	tip["vals"] = vals
	var eff []string
	for _, ex := range gi.Explicits {
		if a, ok := g.affixKey[ex]; ok && a.DescEn != "" {
			eff = append(eff, a.DescEn)
			fb.name(a.DescEn, a.DescIt)
		}
	}
	if len(eff) > 0 {
		tip["eff"] = eff
	}
	if s, ok := g.sets[gi.Set]; ok && gi.Set != "" {
		var pieces [][]any
		have := 0
		for _, p := range s.Pieces {
			on := 0
			if present[p] {
				on, have = 1, have+1
			}
			pn := ""
			if x := getGD().Items[p]; x != nil {
				pn = fb.name(x.En, x.It)
			}
			pieces = append(pieces, []any{pn, on})
		}
		var bon [][]any
		for _, b := range s.Bonuses {
			a := g.affixKey[b.Affix]
			on := 0
			if have >= b.Req {
				on = 1
			}
			fb.name(a.DescEn, a.DescIt)
			bon = append(bon, []any{fmt.Sprintf("(%d) Set:", b.Req), a.DescEn, on})
		}
		tip["set"] = map[string]any{"n": fb.name(s.En, s.It), "pieces": pieces, "bon": bon}
	}
	icon := sa + "Charms/1/" + snakeName(s2(g.sets[gi.Set].En)) + ".png"
	if gi.Type == "HoradricSeal" {
		icon = sa + "Charms/1/seal.png"
	} else if gi.Set == "" {
		icon = sa + "Charms/1/mythic_charm.png"
	}
	return map[string]any{"n": fb.name(en, it), "i": icon, "tip": tip}
}

func s2(s string) string { return s }

// note in formato "lexical" (Maxroll e Mobalytics) → righe come quelle di D4Builds
func lexicalNotes(root any, title string) []any {
	var out []any
	if title != "" {
		out = append(out, []any{map[string]any{"b": title}})
	}
	var line []any
	flush := func() {
		if len(line) > 0 {
			out = append(out, line)
			line = nil
		}
	}
	var walk func(n any)
	walk = func(n any) {
		m, ok := n.(map[string]any)
		if !ok {
			return
		}
		typ, _ := m["type"].(string)
		switch typ {
		case "text":
			t, _ := m["text"].(string)
			f, _ := m["format"].(float64)
			if int(f)&1 == 1 {
				line = append(line, map[string]any{"b": t})
			} else {
				line = append(line, t)
			}
			return
		case "linebreak":
			flush()
			return
		case "embed", "d4-tag", "mention":
			t, _ := m["text"].(string)
			if t == "" {
				t, _ = m["name"].(string)
			}
			et, _ := m["embedType"].(string)
			k := "s"
			if strings.Contains(et, "item") {
				k = "u"
			}
			if t != "" {
				line = append(line, map[string]any{"n": t, "t": k})
			}
			return
		case "heading":
			flush()
			var sb strings.Builder
			var txt func(x any)
			txt = func(x any) {
				if mm, ok := x.(map[string]any); ok {
					if s, ok := mm["text"].(string); ok {
						sb.WriteString(s)
					}
					if ch, ok := mm["children"].([]any); ok {
						for _, c := range ch {
							txt(c)
						}
					}
				}
			}
			txt(m)
			if s := strings.TrimSpace(sb.String()); s != "" {
				out = append(out, []any{map[string]any{"b": s}})
			}
			return
		case "paragraph", "listitem", "quote":
			flush()
			if typ == "listitem" {
				line = append(line, "• ")
			}
			if ch, ok := m["children"].([]any); ok {
				for _, c := range ch {
					walk(c)
				}
			}
			flush()
			return
		}
		if ch, ok := m["children"].([]any); ok {
			for _, c := range ch {
				walk(c)
			}
		}
		if r, ok := m["root"]; ok {
			walk(r)
		}
	}
	walk(root)
	flush()
	return out
}

var slotEnByType = map[string]string{"Helm": "Helm", "ChestArmor": "Chest Armor", "Gloves": "Gloves", "Legs": "Pants", "Boots": "Boots", "Amulet": "Amulet", "Ring": "Ring", "Focus": "Offhand", "OffHandTotem": "Offhand", "Shield": "Offhand", "DaggerOffHand": "Offhand"}

func (fb *fullBuilder) finish(f *D4Full) {
	for en, it := range d4bTranslate(f) {
		if _, ok := fb.tr[en]; !ok {
			fb.tr[en] = it
		}
	}
	f.Tr = fb.tr
	f.OK, f.XV = 1, convXV
}

// ---------------------------------------------------------------- Maxroll
func maxrollFull(id string) (*D4Full, error) {
	g := gxLoad()
	gdd := getGD()
	if g == nil || gdd == nil {
		return nil, fmt.Errorf("dati di gioco non ancora pronti")
	}
	raw, err := plannerRaw(id)
	if err != nil {
		return nil, err
	}
	var top struct {
		Name  string `json:"name"`
		Class string `json:"class"`
		Data  string `json:"data"`
		Date  string `json:"date"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(top.Data), &data); err != nil {
		return nil, err
	}
	base, err := plannerBuild(id)
	if err != nil {
		return nil, err
	}
	fb := &fullBuilder{tr: map[string]string{}}
	items, _ := data["items"].(map[string]any)
	profiles, _ := data["profiles"].([]any)
	cls := strings.ToLower(top.Class)
	treeName := map[string]string{"barbarian": "Barbarian", "druid": "Druid", "necromancer": "Necromancer", "rogue": "Rogue", "sorcerer": "Sorcerer", "spiritborn": "Spiritborn", "paladin": "Paladin_NEW", "warlock": "Warlock"}[cls]
	f := &D4Full{URL: "https://maxroll.gg/d4/planner/" + id, Head: map[string]string{"title": top.Name, "cls": cls, "updated": top.Date}}
	for pi, pv := range profiles {
		p, _ := pv.(map[string]any)
		v := map[string]any{"name": p["name"]}
		var gear, charms, runes []any
		present := map[string]bool{}
		pitems, _ := p["items"].(map[string]any)
		slots := make([]string, 0, len(pitems))
		for k := range pitems {
			slots = append(slots, k)
		}
		sort.Slice(slots, func(a, b int) bool { return atoiSafe(slots[a]) < atoiSafe(slots[b]) })
		for _, sl := range slots {
			it, _ := items[fmt.Sprint(pitems[sl])].(map[string]any)
			if key, _ := it["id"].(string); key != "" {
				present[key] = true
			}
		}
		rings := 0
		var seal map[string]any
		for _, sl := range slots {
			it, _ := items[fmt.Sprint(pitems[sl])].(map[string]any)
			if it == nil {
				continue
			}
			key, _ := it["id"].(string)
			gi := gdd.Items[key]
			if gi == nil {
				continue
			}
			var statsEn, statsIt []string
			var stats []any
			if ex, ok := it["explicits"].([]any); ok {
				for _, e := range ex {
					em, _ := e.(map[string]any)
					nid, _ := em["nid"].(float64)
					a := g.affix[uint32(nid)]
					if a.StatEn == "" {
						continue
					}
					statsEn, statsIt = append(statsEn, a.StatEn), append(statsIt, a.StatIt)
					ga := 0
					if gr, _ := em["greater"].(bool); gr {
						ga = 1
					}
					if up, _ := em["upgrade"].(float64); up > 0 {
						ga = 1
					}
					fb.name(a.StatEn, a.StatIt)
					stats = append(stats, map[string]any{"t": a.StatEn, "ga": ga})
				}
			}
			if gi.Type == "Charm" || gi.Type == "HoradricSeal" {
				c := fb.charm(g, key, gi.En, gi.It, statsEn, statsIt, present)
				if gi.Type == "HoradricSeal" {
					seal = c
				} else {
					charms = append(charms, c)
				}
				continue
			}
			slot := slotEnByType[gi.Type]
			if slot == "" {
				slot = "Weapon"
				if t := gdd.TypesEn[gi.Type]; t != "" {
					slot = t
				}
			}
			if slot == "Ring" {
				rings++
				slot = fmt.Sprintf("Ring %d", rings)
			}
			fb.name(slot, slotIt[slot])
			name, nameIt := gi.En, gi.It
			kind, icon := "", sa+"Uniques/"+map[string]string{"Helm": "helm", "Chest Armor": "chest_armor", "Gloves": "gloves", "Pants": "pants", "Boots": "boots", "Amulet": "amulet", "Ring 1": "ring_1", "Ring 2": "ring_2", "Offhand": "offhand"}[slot]+".png"
			if strings.HasSuffix(icon, "/.png") {
				icon = sa + "Uniques/weapon.png"
			}
			var desc, effs []string
			switch {
			case strings.HasPrefix(key, "Runeword_"):
				kind = "runeword"
			case gi.Mythic:
				kind = "mythic"
			case strings.Contains(key, "_Unique_"):
				kind = "unique"
			}
			if kind != "" {
				icon = sa + "Uniques/2/" + snakeName(gi.En) + ".png"
				for _, ex := range g.items[key].Explicits {
					if a, ok := g.affixKey[ex]; ok && a.DescEn != "" {
						effs = append(effs, a.DescEn)
						fb.name(a.DescEn, a.DescIt)
					}
				}
			}
			if as, ok := it["aspects"].([]any); ok && kind == "" {
				for _, ax := range as {
					am, _ := ax.(map[string]any)
					nid, _ := am["nid"].(float64)
					if ai, ae := aspectName(int(nid)); ae != "" {
						name, nameIt, kind = ae, ai, "legendary"
						icon = sa + "Codex/1/offensive.png"
						if a, ok := g.affix[uint32(nid)]; ok && a.DescEn != "" {
							desc = append(desc, a.DescEn)
							fb.name(a.DescEn, a.DescIt)
						}
						break
					}
				}
			}
			var temper []string
			if tl, ok := it["tempered"].([]any); ok {
				for _, t := range tl {
					tm, _ := t.(map[string]any)
					nid, _ := tm["nid"].(float64)
					if a, ok := g.affix[uint32(nid)]; ok && a.StatEn != "" {
						temper = append(temper, a.StatEn)
						fb.name(a.StatEn, a.StatIt)
					}
				}
			}
			var gems []any
			if sk, ok := it["sockets"].([]any); ok {
				for _, s := range sk {
					k, _ := s.(string)
					x := gdd.Items[k]
					if x == nil {
						continue
					}
					gm := map[string]any{"n": fb.name(x.En, x.It)}
					if strings.HasPrefix(k, "Rune_") {
						ic := sa + "Runes/invocation.png"
						if strings.Contains(k, "Condition") {
							ic = sa + "Runes/ritual.png"
						}
						gm["i"] = ic
						runes = append(runes, map[string]any{"n": x.En, "i": ic})
					} else if ic := gemIconURL(x.En); ic != "" {
						gm["i"] = ic
					}
					gems = append(gems, gm)
				}
			}
			sub := strings.TrimSpace(map[string]string{"mythic": "Mythic Unique", "unique": "Unique", "legendary": "Legendary", "runeword": "Runeword"}[kind] + " " + slot)
			tip := map[string]any{"n": name, "sub": sub, "stats": stats, "temper": temper, "eff": effs}
			if len(desc) > 0 {
				tip["desc"] = desc[0]
			}
			k := kind
			if k == "unique" || k == "runeword" {
				k = "mythic" // come D4Builds: "mythic" nel nome = unico; i veri mitici li riconosce la traduzione
			}
			fb.name(name, nameIt)
			if gi.Mythic {
				fb.tr["~m:"+name] = "1"
			}
			gear = append(gear, map[string]any{"slot": slot, "n": name, "k": k, "i": icon, "gems": gems, "tip": tip})
		}
		v["gear"], v["charms"], v["runes"] = gear, charms, runes
		if seal != nil {
			v["seal"] = seal
		}
		// abilità della barra
		var skills []any
		if sb, ok := p["skillBar"].([]any); ok {
			for _, s := range sb {
				k, _ := s.(string)
				sk := g.skills[k]
				if sk.En == "" {
					continue
				}
				skills = append(skills, map[string]any{"n": fb.name(sk.En, sk.It), "i": sa + "Skills/VoH2/" + snakeName(sk.En) + ".png"})
			}
		}
		v["skills"] = skills
		// albero delle abilità (ultimo passo)
		if st, ok := p["skillTree"].(map[string]any); ok {
			steps, _ := st["steps"].([]any)
			if len(steps) > 0 {
				last, _ := steps[len(steps)-1].(map[string]any)
				ranks := map[int]int{}
				if dl, ok := last["data"].([]any); ok {
					for _, x := range dl {
						n, _ := x.(float64)
						ranks[int(n)]++
					}
				}
				if tv := fb.treeView(g, treeName, ranks, 0.25); tv != nil {
					v["tree"] = tv
				}
			}
		}
		// paragon: tavole Maxroll già pronte
		if pi < len(base.Variants) && len(base.Variants[pi].Paragon) > 0 {
			v["mxParagon"] = base.Variants[pi].Paragon
		}
		// mercenari
		if mc, ok := p["mercenary"].(map[string]any); ok {
			idm, _ := mc["id"].(string)
			sup, _ := mc["support"].(string)
			merc := map[string]any{}
			var slotsM []any
			add := func(label, en, it, icon string) {
				if en != "" {
					slotsM = append(slotsM, map[string]any{"l": label, "n": fb.name(en, it), "i": icon})
				}
			}
			mi := g.mercs[idm]
			add("Mercenary", mi[1], mi[0], mercIcon(mi[1]))
			ms := g.mercs[sup]
			add("Reinforcement", ms[1], ms[0], mercIcon(ms[1]))
			if ss, ok := mc["supportSkills"].([]any); ok {
				labels := []string{"Skill", "Opportunity"}
				for i, x := range ss {
					k, _ := x.(string)
					if sk := g.skills[k]; sk.En != "" && i < 2 {
						add(labels[i], sk.En, sk.It, "")
					}
				}
			}
			merc["slots"] = slotsM
			if tr, ok := mc["tree"].([]any); ok {
				ranks := map[int]int{}
				for _, x := range tr {
					n, _ := x.(float64)
					ranks[int(n)] = 1
				}
				tname := strings.Replace(idm, "MercenaryClass_", "Mercenary_", 1)
				var picks []any
				if t, ok := g.trees[tname]; ok {
					for _, n := range t.Nodes {
						if ranks[n.ID] > 0 {
							sk := g.skills[g.rewards[n.Reward].Power]
							if sk.En != "" {
								picks = append(picks, map[string]any{"n": fb.name(sk.En, sk.It), "i": sa + "Skills/VoH2/" + snakeName(sk.En) + ".png"})
							}
						}
					}
				}
				merc["items"] = picks
			}
			v["merc"] = merc
		}
		// piani di guerra
		if wp, ok := p["warPlans"].([]any); ok && len(wp) > 0 {
			var war []any
			for wi, w := range g.war {
				ranks := map[int]int{}
				if wi < len(wp) {
					n, _ := wp[wi].(float64)
					ranks[int(n)] = 1
				}
				tv := fb.treeView(g, w.Tree, ranks, 0.25)
				has := false
				if tv != nil {
					for _, n := range tv["nodes"].([]any) {
						a := n.([]any)
						if a[2] != "c" && fmt.Sprint(a[7]) == "1" {
							has = true
						}
					}
				}
				e := map[string]any{"n": fb.name(w.En, w.It)}
				if has {
					e["tree"] = tv
				}
				war = append(war, e)
			}
			v["war"] = war
		}
		f.Variants = append(f.Variants, v)
	}
	// note: guida generale e note delle sezioni
	var notes []any
	if gn, ok := data["globalNotes"]; ok {
		notes = append(notes, lexicalNotes(gn, "")...)
	}
	secIt := map[string]string{"equipment": "Equipaggiamento", "skills": "Abilità", "paragon": "Paragon", "rotation": "Rotation", "mercenary": "Mercenari", "talismans": "Talismani"}
	if len(profiles) > 0 {
		last, _ := profiles[len(profiles)-1].(map[string]any)
		if wn, ok := last["widgetNotes"].(map[string]any); ok {
			keys := make([]string, 0, len(wn))
			for k := range wn {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				t := secIt[k]
				if t == "" {
					t = strings.Title(k)
				}
				notes = append(notes, lexicalNotes(wn[k], t)...)
			}
		}
	}
	f.Notes = notes
	f.Filters, f.Planner, f.Src = base.Filters, id, "maxroll"
	fb.finish(f)
	return f, nil
}

func atoiSafe(s string) int {
	n := 0
	fmt.Sscan(s, &n)
	return n
}

func gemIconURL(en string) string {
	l := strings.ToLower(en)
	for _, g := range []string{"ruby", "sapphire", "emerald", "topaz", "amethyst", "diamond", "skull"} {
		if strings.Contains(l, g) {
			return sa + "Gems/grand-" + g + ".png"
		}
	}
	return ""
}

// "Raheir, the Shieldbearer" → icona come su D4Builds
func mercIcon(en string) string {
	l := strings.ToLower(en)
	for k, v := range map[string]string{"raheir": "raheir,_the_shieldbearer", "varyana": "varyana,_the_berserker_crone", "aldkin": "aldkin,_the_cursed_child", "subo": "subo,_the_drunken_archer"} {
		if strings.HasPrefix(l, k) {
			return sa + "Mercenaries/" + v + ".png"
		}
	}
	return ""
}

// ---------------------------------------------------------------- Mobalytics
func mStr(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func mMap(m map[string]any, k string) map[string]any {
	x, _ := m[k].(map[string]any)
	return x
}
func mList(m map[string]any, k string) []any { x, _ := m[k].([]any); return x }

func findAll(o any, pred func(map[string]any) bool, out *[]map[string]any, d int) {
	if d > 40 {
		return
	}
	switch v := o.(type) {
	case map[string]any:
		if pred(v) {
			*out = append(*out, v)
		}
		for _, x := range v {
			findAll(x, pred, out, d+1)
		}
	case []any:
		for _, x := range v {
			findAll(x, pred, out, d+1)
		}
	}
}

// statistica Mobalytics ("critical-strike-damage") → nome inglese e italiano
func (g *gxData) mobaStat(slug string) (string, string) {
	if a, ok := g.statSlug[slug]; ok {
		return a.StatEn, a.StatIt
	}
	for _, suf := range []string{"-multiplier", ""} {
		if a, ok := g.statSlug[strings.TrimSuffix(slug, suf)+"-multiplier"]; ok && suf == "-multiplier" {
			return a.StatEn, a.StatIt
		}
	}
	if i := strings.Index(slug, "-talisman"); i > 0 {
		slug = slug[:i]
		if a, ok := g.statSlug[slug]; ok {
			return a.StatEn, a.StatIt
		}
	}
	w := slugWords(slug)
	return w, ""
}

func mobaFull(u string) (*D4Full, error) {
	g := gxLoad()
	gdd := getGD()
	if g == nil || gdd == nil {
		return nil, fmt.Errorf("dati di gioco non ancora pronti")
	}
	st, err := mobaState(u)
	if err != nil {
		return nil, err
	}
	var docs []map[string]any
	findAll(st, func(m map[string]any) bool { _, ok := m["buildVariants"].(map[string]any); return ok }, &docs, 0)
	if len(docs) == 0 {
		return nil, &httpErr{404}
	}
	doc := docs[0]
	vnames := map[string]string{}
	var vw []map[string]any
	findAll(st, func(m map[string]any) bool { _, ok := m["childrenVariants"].([]any); return ok }, &vw, 0)
	for _, w := range vw {
		for _, x := range mList(w, "childrenVariants") {
			if m, ok := x.(map[string]any); ok {
				vnames[mStr(m, "id")] = mStr(m, "title")
			}
		}
	}
	fb := &fullBuilder{tr: map[string]string{}}
	f := &D4Full{URL: u, Head: map[string]string{"title": mStr(doc, "name")}}
	vals := mList(mMap(doc, "buildVariants"), "values")
	cls := ""
	for vi, vx := range vals {
		v, _ := vx.(map[string]any)
		if v == nil {
			continue
		}
		name := vnames[mStr(v, "id")]
		if name == "" {
			name = "Variante " + itoa(vi+1)
		}
		out := map[string]any{"name": name}
		var gear, charms, runes, prime []any
		var seal map[string]any
		present := map[string]bool{}
		slots := mList(mMap(v, "genericBuilder"), "slots")
		for _, sx := range slots {
			sm, _ := sx.(map[string]any)
			ge := mMap(sm, "gameEntity")
			if ge == nil {
				continue
			}
			if _, gi := itName(mStr(ge, "title")); gi != nil {
				present[gi.Key] = true
			}
		}
		for _, sx := range slots {
			sm, _ := sx.(map[string]any)
			ge := mMap(sm, "gameEntity")
			if ge == nil {
				continue
			}
			en := strings.TrimSpace(mStr(ge, "title"))
			typ := mStr(ge, "type")
			mods := mMap(ge, "modifiers")
			statsOf := func(key string) ([]any, []string, []string) {
				var list []any
				var se, si []string
				for _, x := range mList(mods, key) {
					m, _ := x.(map[string]any)
					sEn, sIt := g.mobaStat(mStr(m, "id"))
					fb.name(sEn, sIt)
					ga := 0
					if m["isGreater"] == true {
						ga = 1
					}
					list = append(list, map[string]any{"t": sEn, "ga": ga})
					se, si = append(se, sEn), append(si, sIt)
				}
				return list, se, si
			}
			switch typ {
			case "charms", "seals":
				_, gi := itName(en)
				key, it := "", en
				if gi != nil {
					key, it = gi.Key, gi.It
				}
				_, se, si := statsOf(map[string]string{"charms": "charmStats", "seals": "sealStats"}[typ])
				c := fb.charm(g, key, en, it, se, si, present)
				if ic := mStr(ge, "iconUrl"); ic != "" && key == "" {
					c["i"] = ic
				}
				if typ == "seals" {
					seal = c
				} else {
					charms = append(charms, c)
				}
				continue
			case "triadPowers":
				prime = append(prime, map[string]any{"n": en, "i": mStr(ge, "iconUrl")})
				continue
			}
			slotSlug := mStr(sm, "gameSlotSlug")
			sn, ok := mobaSlotNames[slotSlug]
			if !ok {
				continue
			}
			slot := sn[1]
			if slotSlug == "ring-1" {
				slot = "Ring 1"
			} else if slotSlug == "ring-2" {
				slot = "Ring 2"
			}
			fb.name(slot, slotIt[slot])
			itn, gi := itName(en)
			if gi == nil {
				itn = trName(en)
			}
			kind := ""
			icon, fbIcon := "", mStr(ge, "iconUrl")
			var effs []string
			desc := ""
			switch typ {
			case "uniqueItems":
				kind = "mythic"
				if e := mMap(ge, "entity"); e != nil && e["mythic"] == true {
					fb.tr["~m:"+en] = "1"
				}
				icon = sa + "Uniques/2/" + snakeName(en) + ".png"
				if gi != nil {
					for _, ex := range g.items[gi.Key].Explicits {
						if a, ok := g.affixKey[ex]; ok && a.DescEn != "" {
							effs = append(effs, a.DescEn)
							fb.name(a.DescEn, a.DescIt)
						}
					}
				}
			case "aspects":
				icon = sa + "Codex/1/offensive.png"
				for id, a := range gdd.Aspects {
					if strings.EqualFold(a[1], en) {
						itn = a[0]
						if x, ok := g.affix[uint32(id)]; ok && x.DescEn != "" {
							desc = x.DescEn
							fb.name(x.DescEn, x.DescIt)
						}
						break
					}
				}
			default:
				icon = fbIcon
			}
			fb.name(en, itn)
			stats, _, _ := statsOf("gearStats")
			if is, _, _ := statsOf("implicitStats"); len(is) > 0 {
				stats = append(is, stats...)
			}
			var temper, hor []string
			for _, x := range mList(mods, "temperingStats") {
				m, _ := x.(map[string]any)
				temper = append(temper, slugWords(mStr(m, "id")))
			}
			for _, k := range []string{"transfiguredStats", "kulleanTransfiguredStats", "horadricStats"} {
				for _, x := range mList(mods, k) {
					m, _ := x.(map[string]any)
					hor = append(hor, slugWords(mStr(m, "id")))
				}
			}
			var gems []any
			for _, x := range mList(mods, "socketStats") {
				m, _ := x.(map[string]any)
				w := slugWords(mStr(m, "slug"))
				it2, gi2 := itName(w)
				if gi2 != nil {
					w = fb.name(gi2.En, it2)
				}
				gm := map[string]any{"n": w}
				if mStr(m, "type") == "runes" {
					rt := mStr(mMap(m, "entity"), "type")
					ic := sa + "Runes/invocation.png"
					if strings.EqualFold(rt, "ritual") {
						ic = sa + "Runes/ritual.png"
					}
					gm["i"] = ic
					runes = append(runes, map[string]any{"n": w, "i": ic})
				} else if ic := gemIconURL(w); ic != "" {
					gm["i"] = ic
				}
				gems = append(gems, gm)
			}
			sub := map[string]string{"uniqueItems": "Unique", "aspects": "Legendary"}[typ] + " " + slot
			tip := map[string]any{"n": en, "sub": strings.TrimSpace(sub), "stats": stats, "temper": temper, "hor": hor, "eff": effs}
			if desc != "" {
				tip["desc"] = desc
			}
			gear = append(gear, map[string]any{"slot": slot, "n": en, "k": kind, "i": icon, "fb": fbIcon, "gems": gems, "tip": tip})
		}
		out["gear"], out["charms"], out["runes"], out["prime"] = gear, charms, dedupeRunes(runes), prime
		if seal != nil {
			out["seal"] = seal
		}
		// barra delle abilità
		as := mMap(v, "assignedSkills")
		type ps struct {
			pos int
			s   map[string]any
		}
		var list []ps
		for _, x := range mList(as, "skills") {
			m, _ := x.(map[string]any)
			sk := mMap(m, "skill")
			if sk == nil {
				continue
			}
			en := mStr(sk, "name")
			if cls == "" {
				cls = strings.ToLower(mStr(sk, "classSlug"))
			}
			pos, _ := m["position"].(float64)
			itn := en
			for _, s := range gdd.Skills {
				if strings.EqualFold(s[1], en) {
					itn = s[0]
					break
				}
			}
			list = append(list, ps{int(pos), map[string]any{"n": fb.name(en, itn), "i": sa + "Skills/VoH2/" + snakeName(en) + ".png"}})
		}
		sort.SliceStable(list, func(a, b int) bool { return list[a].pos < list[b].pos })
		var skills []any
		for _, x := range list {
			skills = append(skills, x.s)
		}
		out["skills"] = skills
		if cls == "" {
			for _, c := range []string{"barbarian", "druid", "necromancer", "rogue", "sorcerer", "spiritborn", "paladin", "warlock"} {
				if strings.Contains(u, c) {
					cls = c
				}
			}
		}
		treeName := map[string]string{"barbarian": "Barbarian", "druid": "Druid", "necromancer": "Necromancer", "rogue": "Rogue", "sorcerer": "Sorcerer", "spiritborn": "Spiritborn", "paladin": "Paladin_NEW", "warlock": "Warlock"}[cls]
		// albero delle abilità: le voci di Mobalytics sono nomi (slug) di abilità e potenziamenti
		if ranks := g.slugRanks(treeName, mList(mMap(v, "skillTree"), "skills")); len(ranks) > 0 {
			if tv := fb.treeView(g, treeName, ranks, 0.25); tv != nil {
				out["tree"] = tv
			}
		}
		// paragon con le tavole posizionate come su Mobalytics
		if par := mobaParagon(fb, gdd, cls, mMap(v, "paragon")); par != nil {
			out["paragon"] = par
		}
		// mercenari
		if mc := mMap(v, "mercenary"); mc != nil {
			var sl []any
			add := func(label, slug string, merc bool) {
				if slug == "" {
					return
				}
				n := slugWords(slug)
				icon := ""
				if merc {
					icon = mercIcon(n)
					for _, m := range g.mercs {
						if strings.HasPrefix(strings.ToLower(n), strings.ToLower(m[1])) && m[1] != "" {
							fb.name(n, m[0]+strings.TrimPrefix(n, strings.Fields(n)[0]))
						}
					}
				} else {
					for _, s := range g.skills {
						if textSlug(s.En) == slug {
							fb.name(n, s.It)
							break
						}
					}
				}
				sl = append(sl, map[string]any{"l": label, "n": n, "i": icon})
			}
			add("Mercenary", mStr(mMap(mc, "primaryMercenary"), "slug"), true)
			add("Reinforcement", mStr(mMap(mc, "reinforcementMercenary"), "slug"), true)
			add("Skill", mStr(mMap(mc, "skill"), "slug"), false)
			add("Opportunity", mStr(mMap(mMap(mc, "opportunity"), "skill"), "slug"), false)
			var items []any
			for _, x := range mList(mc, "skillTree") {
				m, _ := x.(map[string]any)
				slug := mStr(mMap(m, "skill"), "slug")
				n := slugWords(slug)
				for _, s := range g.skills {
					if textSlug(s.En) == slug {
						n = fb.name(s.En, s.It)
						break
					}
				}
				items = append(items, map[string]any{"n": n, "i": sa + "Skills/VoH2/" + snakeName(n) + ".png"})
			}
			out["merc"] = map[string]any{"slots": sl, "items": items}
		}
		// piani di guerra
		if wp := mMap(v, "warPlan"); wp != nil {
			var war []any
			for _, x := range mList(wp, "warplanTrees") {
				m, _ := x.(map[string]any)
				tslug := mStr(mMap(m, "warplanTree"), "slug")
				var tree string
				name, nameIt := slugWords(tslug), ""
				for _, w := range g.war {
					if textSlug(w.En) == tslug || strings.Contains(tslug, textSlug(w.En)) || strings.Contains(textSlug(w.En), strings.TrimPrefix(tslug, "tree-of-")) {
						tree, name, nameIt = w.Tree, w.En, w.It
						break
					}
				}
				e := map[string]any{"n": fb.name(name, nameIt)}
				if ranks := g.slugRanks(tree, mList(m, "warplanNodes")); len(ranks) > 0 {
					e["tree"] = fb.treeView(g, tree, ranks, 0.25)
				}
				war = append(war, e)
			}
			out["war"] = war
		}
		f.Variants = append(f.Variants, out)
	}
	f.Head["cls"] = cls
	// note: testi della guida
	var notes []any
	var rich []map[string]any
	findAll(st, func(m map[string]any) bool { return mStr(m, "__typename") == "NgfDocumentCmWidgetRichTextSimplifiedV2" }, &rich, 0)
	for _, r := range rich {
		d := mMap(r, "data")
		notes = append(notes, lexicalNotes(mMap(mMap(d, "simplifiedContent"), "value"), mStr(d, "title"))...)
	}
	f.Notes = notes
	f.Src = "moba"
	fb.finish(f)
	return f, nil
}

func dedupeRunes(in []any) []any {
	seen := map[string]bool{}
	var out []any
	for _, x := range in {
		m := x.(map[string]any)
		n := fmt.Sprint(m["n"])
		if !seen[n] {
			seen[n] = true
			out = append(out, x)
		}
	}
	return out
}

// ranghi dei nodi di un albero a partire dagli slug di Mobalytics (una voce per ogni punto)
func (g *gxData) slugRanks(tree string, list []any) map[int]int {
	t, ok := g.trees[tree]
	if !ok {
		return nil
	}
	bySlug := map[string]int{}
	for _, n := range t.Nodes {
		r := g.rewards[n.Reward]
		sk := g.skills[r.Power]
		if sk.En == "" {
			continue
		}
		switch r.Type {
		case 1:
			if m, ok := sk.Mods[r.Mod]; ok {
				bySlug[textSlug(sk.En+" "+m[1])] = n.ID
				if _, dup := bySlug[textSlug(m[1])]; !dup {
					bySlug[textSlug(m[1])] = n.ID
				}
			}
		default:
			if _, dup := bySlug[textSlug(sk.En)]; !dup || r.Type == 0 {
				bySlug[textSlug(sk.En)] = n.ID
			}
		}
	}
	ranks := map[int]int{}
	for _, x := range list {
		m, _ := x.(map[string]any)
		slug := mStr(m, "slug")
		if s := mMap(m, "skill"); s != nil {
			slug = mStr(s, "slug")
		}
		if id, ok := bySlug[slug]; ok {
			ranks[id]++
		}
	}
	return ranks
}

var reParNode = regexp.MustCompile(`^(.*)-x(\d+)-y(\d+)$`)

func mobaParagon(fb *fullBuilder, gdd *gameData, cls string, p map[string]any) map[string]any {
	if p == nil {
		return nil
	}
	active := map[string]map[int]bool{}
	for _, x := range mList(p, "nodes") {
		m, _ := x.(map[string]any)
		if mm := reParNode.FindStringSubmatch(mStr(m, "slug")); mm != nil {
			if active[mm[1]] == nil {
				active[mm[1]] = map[int]bool{}
			}
			active[mm[1]][atoiSafe(mm[3])*1000+atoiSafe(mm[2])] = true
		}
	}
	tok := map[string]string{"barbarian": "barb", "druid": "druid", "necromancer": "necro", "rogue": "rogue", "sorcerer": "sorc", "spiritborn": "spirit", "paladin": "paladin", "warlock": "warlock"}[cls]
	var boards []any
	for _, x := range mList(p, "boards") {
		m, _ := x.(map[string]any)
		bslug := mStr(mMap(m, "board"), "slug")
		rest := strings.TrimPrefix(bslug, cls+"-")
		var def PBoardDef
		var key string
		for k, b := range gdd.Boards {
			if !strings.Contains(strings.ToLower(k), tok) {
				continue
			}
			if textSlug(b.En) == rest || rest == "starter-board" && b.En == "Start" {
				def, key = b, k
				break
			}
		}
		if key == "" {
			continue
		}
		w := def.Width
		if w == 0 {
			w = 21
		}
		var tiles []any
		rar := map[byte]string{1: "Common", 2: "Magic", 3: "Rare", 4: "Legendary", 5: "Gate", 6: "Glyph", 7: "Common"}
		rot, _ := m["rotation"].(float64)
		// Mobalytics non dice se le coordinate partono da 0 o da 1, né se sono già ruotate:
		// si sceglie la lettura in cui cadono più nodi presi su caselle che esistono
		q := ((int(rot)/90)%4 + 4) % 4
		mapAt := func(r, c, off, rq int) int {
			for k := 0; k < rq; k++ { // posizione della casella dopo la rotazione oraria della tavola
				r, c = c, w-1-r
			}
			return (r+off)*1000 + (c + off)
		}
		bestOff, bestRq, bestHit := 0, 0, -1
		for _, off := range []int{0, 1} {
			for _, rq := range []int{0, q} {
				hit := 0
				for i, c := range def.Cells {
					if c != 0 && active[bslug][mapAt(i/w, i%w, off, rq)] {
						hit++
					}
				}
				if hit > bestHit {
					bestOff, bestRq, bestHit = off, rq, hit
				}
			}
		}
		for i, c := range def.Cells {
			if c == 0 {
				continue
			}
			r, col := i/w, i%w
			on := 0
			if active[bslug][mapAt(r, col, bestOff, bestRq)] {
				on = 1
			}
			name := ""
			if i < len(def.Names) {
				name = def.Names[i]
			}
			tiles = append(tiles, []any{r, col, on, name, "", rar[c]})
		}
		gl := mStr(mMap(m, "glyph"), "slug")
		glyph := ""
		for _, gg := range gdd.Glyphs {
			if textSlug(gg[1]) == strings.TrimPrefix(gl, cls+"-") {
				glyph = fb.name(gg[1], gg[0])
				break
			}
		}
		bx, _ := m["x"].(float64)
		by, _ := m["y"].(float64)
		boards = append(boards, map[string]any{"n": fb.name(def.En, def.It), "g": glyph, "x": int(bx) * 1258, "y": int(by) * 1258, "r": int(rot), "tiles": tiles})
	}
	if len(boards) == 0 {
		return nil
	}
	return map[string]any{"boards": boards}
}

// build completa di qualunque fonte: prima dal server, poi (se si può) letta e convertita qui
// versione della conversione Maxroll/Mobalytics → formato D4Builds: va alzata quando cambia,
// così il raccoglitore le rifà e le app non usano le copie vecchie
const convXV = d4bExtractVer + 100*2

func fullXV(u string) int {
	if strings.HasPrefix(u, "https://d4builds.gg/") {
		return d4bExtractVer
	}
	return convXV
}

func hasXV(b []byte, v int) bool {
	return bytes.Contains(b, []byte(fmt.Sprintf(`"xv":%d,`, v))) || bytes.Contains(b, []byte(fmt.Sprintf(`"xv":%d}`, v)))
}

func anyFull(u, planner string) ([]byte, error) {
	if planner != "" {
		if !plannerID.MatchString(planner) {
			return nil, &httpErr{400}
		}
		p := cacheFile("mxfull_" + planner + ".json")
		if b, err := os.ReadFile(p); err == nil && fresh(p, 6*time.Hour) && hasXV(b, convXV) {
			return b, nil
		}
		f, err := maxrollFull(planner)
		if err != nil {
			if b, e2 := os.ReadFile(p); e2 == nil {
				return b, nil
			}
			return nil, err
		}
		b, _ := json.Marshal(f)
		os.WriteFile(p, b, 0o644)
		return b, nil
	}
	if b := serverFull(u); b != nil && (hasXV(b, fullXV(u)) || !isCollector()) {
		return b, nil
	}
	if !isCollector() {
		return nil, fmt.Errorf("questa build arriverà con il prossimo aggiornamento dei dati")
	}
	var f *D4Full
	var err error
	switch {
	case strings.HasPrefix(u, "https://d4builds.gg/builds/"):
		f, err = d4bFullRead(u)
	case strings.HasPrefix(u, "https://mobalytics.gg/diablo-4/"):
		f, err = mobaFull(u)
		if err == nil {
			f.URL, f.At = u, time.Now().Unix()
		}
	default:
		return nil, &httpErr{400}
	}
	if err != nil {
		if b := serverFull(u); b != nil {
			return b, nil
		}
		return nil, err
	}
	go uploadFull(u, f)
	return json.Marshal(f)
}

func fresh(p string, d time.Duration) bool {
	st, err := os.Stat(p)
	return err == nil && time.Since(st.ModTime()) < d
}
