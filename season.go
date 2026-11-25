package main

// Elaborazione dei dati di una stagione: build per classe (con la tier list di Rob) e
// indice "chi droppa cosa" con i nomi italiani.

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type robRow struct {
	Tier     string `json:"tier"`
	TierNote string `json:"tierNote"`
	Build    string `json:"build"`
	Info     string `json:"info"`
	URL      string `json:"url"`
	Cls      string `json:"cls"`
	Rank     int    `json:"rank"`
}

type seasonRaw struct {
	Season  int `json:"season"`
	Maxroll []struct {
		Slug     string `json:"slug"`
		Title    string `json:"title"`
		Cls      string `json:"cls"`
		Type     string `json:"type"`
		Author   string `json:"author"`
		Updated  int64  `json:"updated"`
		URL      string `json:"url"`
		Profile  string `json:"profile"`
		Desc     string `json:"desc"`
		Leveling string `json:"leveling"`
	} `json:"maxroll"`
	D4builds []struct {
		Slug, Name, Cls, Type, URL string
	} `json:"d4builds"`
	Mobalytics []struct {
		Name     string   `json:"name"`
		Cls      string   `json:"cls"`
		Type     string   `json:"type"`
		Author   string   `json:"author"`
		URL      string   `json:"url"`
		Verified bool     `json:"verified"`
		Likes    int      `json:"likes"`
		Updated  int64    `json:"updated"`
		Activity []string `json:"activity"`
	} `json:"mobalytics"`
	Rob struct {
		Sheet         string   `json:"sheet"`
		LevelingSheet string   `json:"levelingSheet"`
		WIP           bool     `json:"wip"`
		Source        string   `json:"source"`
		Tier          []robRow `json:"tier"`
		Leveling      []robRow `json:"leveling"`
	} `json:"rob"`
	Loot struct {
		Source string `json:"source"`
		PTR    bool   `json:"ptr"`
		Bosses []struct {
			Boss     string     `json:"boss"`
			Name     string     `json:"name"`
			Location string     `json:"location"`
			All      []string   `json:"all"`
			ByClass  classLists `json:"byClass"`
		} `json:"bosses"`
	} `json:"loot"`
	LootMaxroll struct {
		Sections []struct {
			Name  string `json:"name"`
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"sections"`
	} `json:"lootMaxroll"`
	BossInfo struct {
		Source string `json:"source"`
		URL    string `json:"url"`
		Bosses []struct {
			Name  string   `json:"name"`
			Lines []string `json:"lines"`
		} `json:"bosses"`
	} `json:"bossInfo"`
	Updated map[string]int64 `json:"updated"`
}

type BuildEntry struct {
	Src      string `json:"src"`
	Title    string `json:"title"`
	Cls      string `json:"cls"`
	Type     string `json:"type"`
	Author   string `json:"author,omitempty"`
	URL      string `json:"url"`
	Profile  string `json:"profile,omitempty"`
	Updated  int64  `json:"updated,omitempty"`
	Tier     string `json:"tier,omitempty"`
	TierNote string `json:"tierNote,omitempty"`
	Info     string `json:"info,omitempty"`
	Rank     int    `json:"rank,omitempty"`
	Likes    int    `json:"likes,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

type LootSource struct {
	Boss     string `json:"boss"`
	Location string `json:"location,omitempty"`
	Cls      string `json:"cls,omitempty"`
	Kind     string `json:"kind"` // boss | special | mythic
}

type LootItem struct {
	Name    string       `json:"name"`
	En      string       `json:"en"`
	Key     string       `json:"key,omitempty"`
	Type    string       `json:"type,omitempty"`
	Classes []string     `json:"classes,omitempty"`
	Mythic  bool         `json:"mythic,omitempty"`
	Mat     bool         `json:"mat,omitempty"` // materiale di evocazione / runa
	Cat     string       `json:"cat"`           // tipologia: armi, elmi, torso, guanti, pantaloni, stivali, amuleti, anelli, talismani, altro
	Sources []LootSource `json:"sources"`
}

type LootBoss struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Location string   `json:"location,omitempty"`
	Kind     string   `json:"kind"`
	Items    []int    `json:"items"` // indici in Items
	Summon   []string `json:"summon,omitempty"`
	Info     []string `json:"info,omitempty"`
}

type SeasonView struct {
	Season  int              `json:"season"`
	Builds  []BuildEntry     `json:"builds"`
	Rob     any              `json:"rob"`
	Items   []*LootItem      `json:"items"`
	Bosses  []*LootBoss      `json:"bosses"`
	LootSrc []string         `json:"lootSrc"`
	PTR     bool             `json:"ptr"`
	Updated map[string]int64 `json:"updated"`
}

var views struct {
	sync.Mutex
	m map[int]*SeasonView
}

func invalidateSeason(n int) {
	views.Lock()
	if n == 0 {
		views.m = nil
	} else if views.m != nil {
		delete(views.m, n)
	}
	views.Unlock()
}

func seasonView(n int) *SeasonView {
	views.Lock()
	defer views.Unlock()
	if views.m == nil {
		views.m = map[int]*SeasonView{}
	}
	if v := views.m[n]; v != nil {
		return v
	}
	raw := seasonBytes(n)
	if raw == nil {
		return nil
	}
	var s seasonRaw
	if json.Unmarshal(raw, &s) != nil {
		return nil
	}
	// Mobalytics scaricato dal PC (il server riceve 403)
	mc := mobaSnapshot()
	live := remoteLive()
	if len(s.Mobalytics) == 0 {
		for _, b := range mc.Builds {
			if b.Season == n || (b.Season == 0 && n == live) {
				s.Mobalytics = append(s.Mobalytics, struct {
					Name     string   `json:"name"`
					Cls      string   `json:"cls"`
					Type     string   `json:"type"`
					Author   string   `json:"author"`
					URL      string   `json:"url"`
					Verified bool     `json:"verified"`
					Likes    int      `json:"likes"`
					Updated  int64    `json:"updated"`
					Activity []string `json:"activity"`
				}{b.Name, b.Cls, b.Type, b.Author, b.URL, b.Verified, b.Likes, b.Updated, b.Activity})
			}
		}
	}
	if len(s.BossInfo.Bosses) == 0 && len(mc.Bosses) > 0 && n <= live+1 {
		s.BossInfo.Source = "mobalytics.gg"
		for _, b := range mc.Bosses {
			s.BossInfo.Bosses = append(s.BossInfo.Bosses, struct {
				Name  string   `json:"name"`
				Lines []string `json:"lines"`
			}{b.Name, b.Lines})
		}
	}
	// stagione nuova senza tabelle loot (non ancora uscita): usa quelle della precedente
	prevLoot := false
	if len(s.Loot.Bosses) == 0 && len(s.LootMaxroll.Sections) == 0 {
		if pr := seasonBytes(n - 1); pr != nil {
			var p seasonRaw
			if json.Unmarshal(pr, &p) == nil {
				s.Loot, s.LootMaxroll = p.Loot, p.LootMaxroll
				prevLoot = len(p.Loot.Bosses) > 0 || len(p.LootMaxroll.Sections) > 0
			}
		}
	}
	v := buildView(&s)
	if prevLoot {
		v.LootSrc = append([]string{"stagione " + itoa(n-1) + " (la nuova non ha ancora tabelle)"}, v.LootSrc...)
	}
	views.m[n] = v
	return v
}

var wordRe = regexp.MustCompile(`[a-z0-9]+`)
var stopWords = map[string]bool{"barb": true, "barbarian": true, "druid": true, "necro": true, "necromancer": true, "rogue": true, "sorc": true, "sorcerer": true,
	"spiritborn": true, "paladin": true, "pala": true, "pal": true, "warlock": true, "build": true, "guide": true, "endgame": true, "leveling": true, "the": true, "of": true,
	"and": true, "season": true, "self": true, "cast": true, "new": true, "pog": true}

func keyWords(s string) []string {
	var out []string
	for _, w := range wordRe.FindAllString(strings.ToLower(s), -1) {
		if !stopWords[w] && len(w) > 1 {
			out = append(out, w)
		}
	}
	return out
}

func slugOf(u string) string {
	u = strings.Split(u, "?")[0]
	u = strings.TrimSuffix(u, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return u[i+1:]
	}
	return u
}

// la riga di Rob che corrisponde a una build (stesso link D4Builds, oppure tutte le parole chiave nel titolo)
func robMatch(rows []robRow, cls, title, url string) *robRow {
	ts := map[string]bool{}
	for _, w := range keyWords(title) {
		ts[w] = true
	}
	var best *robRow
	for i := range rows {
		r := &rows[i]
		if r.Cls != "" && cls != "" && r.Cls != cls {
			continue
		}
		if url != "" && r.URL != "" && slugOf(r.URL) == slugOf(url) {
			return r
		}
		kw := keyWords(r.Build)
		if len(kw) == 0 {
			continue
		}
		ok := true
		for _, w := range kw {
			ok = ok && ts[w]
		}
		if ok && (best == nil || len(keyWords(best.Build)) < len(kw)) {
			best = r
		}
	}
	return best
}

func buildView(s *seasonRaw) *SeasonView {
	v := &SeasonView{Season: s.Season, Updated: s.Updated, PTR: s.Loot.PTR}
	rob := s.Rob.Tier
	robLv := s.Rob.Leveling
	tierFor := func(b *BuildEntry) {
		rows := rob
		if strings.EqualFold(b.Type, "leveling") {
			rows = robLv
		}
		if r := robMatch(rows, b.Cls, b.Title, b.URL); r != nil {
			b.Tier, b.TierNote, b.Rank = r.Tier, r.TierNote, r.Rank
			if b.Info == "" {
				b.Info = r.Info
			}
		}
	}
	for _, m := range s.Maxroll {
		b := BuildEntry{Src: "Maxroll", Title: m.Title, Cls: m.Cls, Type: m.Type, Author: m.Author, URL: m.URL, Profile: m.Profile, Updated: m.Updated, Desc: m.Desc}
		tierFor(&b)
		v.Builds = append(v.Builds, b)
	}
	seenD4 := map[string]bool{}
	for _, d := range s.D4builds {
		b := BuildEntry{Src: "D4Builds", Title: d.Name, Cls: d.Cls, Type: d.Type, URL: d.URL}
		tierFor(&b)
		seenD4[d.Slug] = true
		v.Builds = append(v.Builds, b)
	}
	for _, m := range s.Mobalytics {
		b := BuildEntry{Src: "Mobalytics", Title: m.Name, Cls: m.Cls, Type: m.Type, Author: m.Author, URL: m.URL, Likes: m.Likes, Verified: m.Verified, Updated: m.Updated}
		tierFor(&b)
		v.Builds = append(v.Builds, b)
	}
	// righe di Rob con un planner D4Builds non già in elenco (le sue build personali)
	for li, list := range [][]robRow{rob, robLv} {
		for _, r := range list {
			if r.URL == "" || !strings.Contains(r.URL, "d4builds.gg") || seenD4[slugOf(r.URL)] {
				continue
			}
			typ := "Endgame"
			if strings.Contains(r.URL, "leveling") || li == 1 {
				typ = "Leveling"
			}
			seenD4[slugOf(r.URL)] = true
			v.Builds = append(v.Builds, BuildEntry{Src: "D4Builds", Title: r.Build, Cls: r.Cls, Type: typ, URL: r.URL, Author: "Rob2628", Tier: r.Tier, TierNote: r.TierNote, Info: r.Info, Rank: r.Rank})
		}
	}
	v.Rob = s.Rob
	buildLoot(v, s)
	return v
}

var skipTypes = map[string]bool{"BossSummoning": true, "ConditionRune": true, "EffectRune": true}

func bossID(name string) string {
	n := strings.ToLower(name)
	for _, k := range []string{"varshan", "grigoire", "beast", "zir", "urivar", "duriel", "andariel", "harbinger", "butcher", "astaroth", "bartuc", "belial", "lilith", "mephisto", "baal", "diablo"} {
		if strings.Contains(n, k) {
			return k
		}
	}
	return strings.Join(keyWords(n), "-")
}

func buildLoot(v *SeasonView, s *seasonRaw) {
	byKey := map[string]*LootItem{}
	bosses := map[string]*LootBoss{}
	var order []string
	getBoss := func(id, name, loc, kind string) *LootBoss {
		b := bosses[id]
		if b == nil {
			b = &LootBoss{ID: id, Name: name, Location: loc, Kind: kind}
			bosses[id] = b
			order = append(order, id)
		}
		if b.Location == "" {
			b.Location = loc
		}
		return b
	}
	addItem := func(name string, id int, src LootSource, b *LootBoss) {
		var g *GItem
		d := getGD()
		if d != nil && id != 0 {
			g = d.ByID[id]
		}
		it := name
		if g == nil {
			it, g = itName(name)
		} else if g.It != "" {
			it = g.It
		}
		key := normName(name)
		if g != nil {
			key = g.Key
		}
		li := byKey[key]
		if li == nil {
			li = &LootItem{Name: it, En: name}
			if g != nil {
				li.Key, li.Classes, li.Mythic = g.Key, g.Classes, g.Mythic
				if g.En != "" {
					li.En = g.En
				}
				if d != nil {
					li.Type = d.Types[g.Type]
				}
				li.Mat = skipTypes[g.Type] || g.Magic < 2 && g.Type != ""
				li.Cat = itemCat(g.Type, g.Key)
			}
			if li.Cat == "" {
				li.Cat = "altro"
			}
			byKey[key] = li
		}
		if src.Kind == "mythic" {
			li.Mythic = true
		}
		dup := false
		for _, x := range li.Sources {
			if x.Boss == src.Boss && x.Cls == src.Cls {
				dup = true
			}
		}
		if !dup {
			li.Sources = append(li.Sources, src)
		}
		if b != nil {
			if li.Mat {
				for _, x := range b.Summon {
					if x == li.Name {
						return
					}
				}
				b.Summon = append(b.Summon, li.Name)
			}
		}
	}
	if len(s.Loot.Bosses) > 0 {
		v.LootSrc = append(v.LootSrc, s.Loot.Source)
	}
	for _, lb := range s.Loot.Bosses {
		if strings.EqualFold(lb.Boss, "All Bosses") {
			b := getBoss("all", "Tutti i boss dei covi", "", "special")
			for _, n := range lb.All {
				addItem(n, 0, LootSource{Boss: b.ID, Kind: "special"}, b)
			}
			continue
		}
		b := getBoss(bossID(lb.Name), lb.Name, lb.Location, "boss")
		for _, n := range lb.All {
			addItem(n, 0, LootSource{Boss: b.ID, Location: lb.Location, Kind: "boss"}, b)
		}
		for cls, list := range lb.ByClass {
			for _, n := range list {
				addItem(n, 0, LootSource{Boss: b.ID, Location: lb.Location, Cls: cls, Kind: "boss"}, b)
			}
		}
	}
	if len(s.LootMaxroll.Sections) > 0 {
		v.LootSrc = append(v.LootSrc, "maxroll.gg")
	}
	for _, sec := range s.LootMaxroll.Sections {
		low := strings.ToLower(sec.Name)
		var b *LootBoss
		kind := "boss"
		switch {
		case strings.Contains(low, "mythic"):
			kind = "mythic"
			b = getBoss("mythic", "Mitici (tutti i boss, raro)", "", "mythic")
		case strings.Contains(low, "non-boss"):
			kind = "special"
			b = getBoss("random", "Nessun boss: drop casuale o Cubo Horadrico", "", "special")
		case strings.Contains(low, "trophies"):
			kind = "special"
			b = getBoss("trophy", "Trofei dei boss (Cubo Horadrico)", "", "special")
		case strings.Contains(low, "new season") || strings.Contains(low, "uniques"):
			kind = "special"
			b = getBoss("capstone", sec.Name+" (Baal, Mefisto, Diablo)", "Dungeon capstone", "special")
		default:
			b = getBoss(bossID(sec.Name), sec.Name, "", "boss")
		}
		for _, x := range sec.Items {
			addItem(x.Name, x.ID, LootSource{Boss: b.ID, Location: b.Location, Kind: kind}, b)
		}
	}
	// indici
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return byKey[keys[i]].Name < byKey[keys[j]].Name })
	idx := map[*LootItem]int{}
	for _, k := range keys {
		idx[byKey[k]] = len(v.Items)
		v.Items = append(v.Items, byKey[k])
	}
	for _, it := range v.Items {
		for _, src := range it.Sources {
			b := bosses[src.Boss]
			if b == nil || it.Mat {
				continue
			}
			i := idx[it]
			dup := false
			for _, x := range b.Items {
				dup = dup || x == i
			}
			if !dup {
				b.Items = append(b.Items, i)
			}
		}
	}
	for _, bi := range s.BossInfo.Bosses {
		if b := bosses[bossID(bi.Name)]; b != nil && b.Kind == "boss" {
			b.Info = bi.Lines
		}
	}
	if len(s.BossInfo.Bosses) > 0 {
		v.LootSrc = append(v.LootSrc, "mobalytics.gg")
	}
	for _, id := range order {
		v.Bosses = append(v.Bosses, bosses[id])
	}
}

// PHP manda [] al posto di {} quando la mappa è vuota
type classLists map[string][]string

func (m *classLists) UnmarshalJSON(b []byte) error {
	var x map[string][]string
	if json.Unmarshal(b, &x) == nil {
		*m = x
	} else {
		*m = classLists{}
	}
	return nil
}

// tipologia di un oggetto dal suo tipo nei dati di gioco
func itemCat(typ, key string) string {
	t := strings.ToLower(typ + " " + key)
	switch {
	case strings.Contains(t, "talisman") || strings.Contains(t, "charm") || strings.Contains(t, "seal"):
		return "talismani"
	case strings.Contains(t, "helm"):
		return "elmi"
	case strings.Contains(t, "chest"):
		return "torso"
	case strings.Contains(t, "glove"):
		return "guanti"
	case strings.Contains(t, "legs") || strings.Contains(t, "pants"):
		return "pantaloni"
	case strings.Contains(t, "boots"):
		return "stivali"
	case strings.Contains(t, "amulet"):
		return "amuleti"
	case strings.Contains(t, "ring"):
		return "anelli"
	case strings.Contains(t, "shield") || strings.Contains(t, "focus") || strings.Contains(t, "totem") || strings.Contains(t, "offhand") || strings.Contains(t, "tome"):
		return "mano secondaria"
	}
	for _, w := range []string{"axe", "sword", "mace", "polearm", "bow", "crossbow", "staff", "wand", "dagger", "scythe", "glaive", "quarterstaff", "flail", "1h", "2h", "weapon"} {
		if strings.Contains(t, w) {
			return "armi"
		}
	}
	return "altro"
}
