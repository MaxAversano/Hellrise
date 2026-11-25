package main

// Filtro loot del gioco generato dalla build: stesso formato dei codici da importare in Diablo IV
// (base64 di un messaggio protobuf: regole con nome, azione, colore e condizioni su tipi, unici, affissi e set).

import (
	"encoding/base64"
	"encoding/binary"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type lfData struct {
	types   map[string][]uint32 // slot inglese (come su D4Builds) → id tipi oggetto
	affix   map[string]uint32   // testo dell'affisso normalizzato → id
	sets    map[string]uint32   // nome del set (inglese) → id
	charm   []uint32            // talismano + sigillo
	weapons []uint32
}

var lfCache struct {
	sync.Mutex
	d   *gameData
	val *lfData
}

var lfClean = regexp.MustCompile(`\{[^}]*\}|\[[^\]]*\]|[+#%x×]`)

func lfNorm(s string) string {
	s = strings.ToLower(lfClean.ReplaceAllString(s, ""))
	s = strings.TrimPrefix(strings.TrimSpace(strings.Join(strings.Fields(s), " ")), "ranks ")
	return s
}

func lfLoad() *lfData {
	d := getGD()
	lfCache.Lock()
	defer lfCache.Unlock()
	if lfCache.val != nil && lfCache.d == d {
		return lfCache.val
	}
	var mn struct {
		ItemTypes map[string]struct {
			ID uint32 `json:"id"`
		} `json:"itemTypes"`
		ItemSets map[string]struct {
			ID uint32 `json:"id"`
		} `json:"itemSets"`
		Affixes map[string]struct {
			ID         uint32 `json:"id"`
			MagicType  int    `json:"magicType"`
			Weight     int    `json:"weight"`
			Attributes []struct {
				ID    int  `json:"id"`
				Param *int `json:"param"`
			} `json:"attributes"`
		} `json:"affixes"`
		Attributes map[string]struct {
			Name string `json:"name"`
		} `json:"attributes"`
	}
	var en struct {
		ItemSets map[string]struct {
			Name string `json:"name"`
		} `json:"itemSets"`
		AttrDesc map[string]string `json:"attributeDescriptions"`
	}
	if readJSON(cacheFile("data.min.json"), &mn) != nil || readJSON(cacheFile("data.enus.json"), &en) != nil {
		return nil
	}
	v := &lfData{types: map[string][]uint32{}, affix: map[string]uint32{}, sets: map[string]uint32{}}
	t := func(names ...string) []uint32 {
		var out []uint32
		for _, n := range names {
			if x, ok := mn.ItemTypes[n]; ok {
				out = append(out, x.ID)
			}
		}
		return out
	}
	v.weapons = t("Mace", "Mace2H", "Sword", "Sword2H", "Axe", "Axe2H", "Staff", "Scythe", "Scythe2H", "Dagger", "Polearm", "Wand", "Bow", "Crossbow", "Crossbow2H", "Flail", "Glaive", "Quarterstaff")
	v.types["helm"] = t("Helm")
	v.types["chest armor"] = t("ChestArmor")
	v.types["gloves"] = t("Gloves")
	v.types["pants"] = t("Legs")
	v.types["boots"] = t("Boots")
	v.types["amulet"] = t("Amulet")
	v.types["ring"] = t("Ring")
	v.types["offhand"] = t("Focus", "OffHandTotem", "Shield", "DaggerOffHand")
	v.types["bludgeoning weapon"] = t("Mace2H", "Flail")
	v.types["slashing weapon"] = t("Sword2H", "Axe2H", "Polearm", "Scythe2H")
	v.types["ranged weapon"] = t("Bow", "Crossbow", "Crossbow2H")
	v.charm = t("Charm", "HoradricSeal")
	for k, s := range mn.ItemSets {
		if n := strings.ToLower(cleanName(en.ItemSets[k].Name)); n != "" {
			v.sets[n] = s.ID
		}
	}
	// affissi: si preferiscono quelli della stagione (S04_/X2_) e con probabilità di uscire
	type cand struct {
		key string
		id  uint32
		pri int
	}
	best := map[string]cand{}
	for k, a := range mn.Affixes {
		if a.MagicType != 0 || len(a.Attributes) == 0 || a.Weight <= 0 {
			continue
		}
		at := a.Attributes[0]
		name := mn.Attributes[itoa(at.ID)].Name
		desc := ""
		if at.Param != nil {
			desc = en.AttrDesc[name+"#"+itoa(*at.Param)]
		}
		if desc == "" {
			desc = en.AttrDesc[name]
		}
		n := lfNorm(desc)
		if n == "" || strings.Contains(n, "value") {
			continue
		}
		pri := 0
		switch {
		case strings.Contains(k, "Greater") || strings.Contains(k, "UBERUNIQUE") || strings.Contains(k, "Transfiguration") || strings.Contains(k, "Talisman"):
			pri = 3
		case strings.HasPrefix(k, "S04_"):
			pri = 0
		case strings.HasPrefix(k, "X2_"):
			pri = 1
		default:
			pri = 2
		}
		if c, ok := best[n]; !ok || pri < c.pri || pri == c.pri && k < c.key {
			best[n] = cand{k, a.ID, pri}
		}
	}
	for n, c := range best {
		v.affix[n] = c.id
	}
	lfCache.d, lfCache.val = d, v
	return v
}

// --- scrittura protobuf ---
type pbuf []byte

func (b pbuf) varint(field int, v uint64) pbuf {
	b = binary.AppendUvarint(b, uint64(field<<3))
	return binary.AppendUvarint(b, v)
}
func (b pbuf) fixed(field int, v uint32) pbuf {
	b = binary.AppendUvarint(b, uint64(field<<3|5))
	return binary.LittleEndian.AppendUint32(b, v)
}
func (b pbuf) bytes(field int, v []byte) pbuf {
	b = binary.AppendUvarint(b, uint64(field<<3|2))
	b = binary.AppendUvarint(b, uint64(len(v)))
	return append(b, v...)
}

type lfCond struct {
	Type int
	IDs  []uint32
	Min  int
}
type lfRule struct {
	Name   string
	Action int // 2 = colora, 3 = nascondi
	Color  uint32
	Conds  []lfCond
}

func lfEncode(name string, rules []lfRule) string {
	var f pbuf
	for _, r := range rules {
		var rb pbuf
		rb = rb.bytes(1, []byte(r.Name)).varint(2, uint64(r.Action)).fixed(3, r.Color)
		for _, c := range r.Conds {
			var cb pbuf
			cb = cb.varint(1, uint64(c.Type))
			for _, id := range c.IDs {
				cb = cb.fixed(2, id)
			}
			if c.Min > 0 {
				cb = cb.varint(4, uint64(c.Min))
			}
			rb = rb.bytes(4, cb)
		}
		rb = rb.varint(5, 1)
		f = f.bytes(1, rb)
	}
	f = f.bytes(2, []byte(name)).varint(3, 1).varint(4, 4)
	return base64.StdEncoding.EncodeToString(f)
}

// richiesta dall'app: cosa serve alla build
type LFReq struct {
	Name    string   `json:"name"`
	Uniques []string `json:"uniques"` // unici e mitici (anche talismani unici), nomi inglesi
	Sets    []string `json:"sets"`    // set di talismani
	Slots   []struct {
		Slot  string   `json:"slot"`
		Stats []string `json:"stats"`
	} `json:"slots"`
	MinAffix int  `json:"minAffix"`
	Hide     bool `json:"hide"`
}

type LFOut struct {
	Code    string   `json:"code"`
	Rules   []string `json:"rules"`
	Missing []string `json:"missing"` // statistiche che il gioco non riconosce per nome
}

func buildLootFilter(q LFReq) LFOut {
	out := LFOut{}
	v := lfLoad()
	if v == nil {
		return out
	}
	const (
		cMythic = 0xFFC07BFF
		cUnique = 0xFFE8B56A
		cGear   = 0xFFFF8A3D
		cCharm  = 0xFF7FE0C0
	)
	var rules []lfRule
	rules = append(rules, lfRule{"Mitici", 2, cMythic, []lfCond{{Type: 1, Min: 32}}})
	var uniq []uint32
	seen := map[uint32]bool{}
	for _, n := range q.Uniques {
		if _, g := itName(n); g != nil && !seen[uint32(g.ID)] {
			seen[uint32(g.ID)] = true
			uniq = append(uniq, uint32(g.ID))
		}
	}
	if len(uniq) > 0 {
		rules = append(rules, lfRule{"Unici della build", 2, cUnique, []lfCond{{Type: 8, IDs: uniq}}})
	}
	for _, s := range q.Sets {
		if id, ok := v.sets[strings.ToLower(s)]; ok {
			rules = append(rules, lfRule{"Set: " + s, 2, cCharm, []lfCond{{Type: 9, IDs: []uint32{id}}, {Type: 5, IDs: v.charm}}})
		}
	}
	minA := q.MinAffix
	if minA <= 0 {
		minA = 2
	}
	missing := map[string]bool{}
	doneSlot := map[string]bool{}
	for _, sl := range q.Slots {
		key := strings.ToLower(strings.TrimSpace(sl.Slot))
		key = strings.TrimSuffix(strings.TrimSuffix(key, " 1"), " 2")
		if strings.HasPrefix(key, "dual-wield weapon") || key == "weapon" {
			key = "weapon"
		}
		if doneSlot[key] {
			continue
		}
		types := v.types[key]
		if key == "weapon" {
			types = v.weapons
		}
		if len(types) == 0 {
			continue
		}
		var ids []uint32
		got := map[uint32]bool{}
		for _, st := range sl.Stats {
			n := lfNorm(regexp.MustCompile(`\s*\(.*\)$`).ReplaceAllString(st, ""))
			id, ok := v.affix[n]
			if !ok {
				missing[st] = true
				continue
			}
			if !got[id] {
				got[id] = true
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		doneSlot[key] = true
		m := minA
		if m > len(ids) {
			m = len(ids)
		}
		label := slotIt[strings.Title(key)]
		if label == "" {
			label = map[string]string{"weapon": "Arma", "ring": "Anello", "offhand": "Mano secondaria"}[key]
		}
		if label == "" {
			label = key
		}
		rules = append(rules, lfRule{label + " con gli affissi giusti", 2, cGear, []lfCond{{Type: 5, IDs: types}, {Type: 6, IDs: ids, Min: m}}})
	}
	if q.Hide {
		var all []uint32
		for _, t := range v.types {
			all = append(all, t...)
		}
		all = append(all, v.weapons...)
		sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
		rules = append(rules, lfRule{"Nascondi il resto", 3, 0xFFFF0000, []lfCond{{Type: 5, IDs: all}}})
	}
	name := q.Name
	if len(name) > 40 {
		name = name[:40]
	}
	out.Code = lfEncode(name, rules)
	for _, r := range rules {
		out.Rules = append(out.Rules, r.Name)
	}
	for k := range missing {
		out.Missing = append(out.Missing, k)
	}
	sort.Strings(out.Missing)
	return out
}
