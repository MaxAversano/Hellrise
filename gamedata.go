package main

// Dati di gioco di Maxroll (nomi in italiano di oggetti, aspetti, abilità) e mappa di Sanctuarium.
// Scaricati dalla CDN di Maxroll e tenuti sul PC; si riscaricano una volta al giorno se cambiano.

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const mxCDN = "https://assets-ng.maxroll.gg/d4-tools/game/"

type GItem struct {
	Key     string   `json:"key"`
	ID      int      `json:"id"`
	Type    string   `json:"type"`
	Magic   int      `json:"magic"`
	It      string   `json:"it"`
	En      string   `json:"en"`
	Classes []string `json:"classes,omitempty"`
	Mythic  bool     `json:"mythic,omitempty"`
}

type gameData struct {
	Version   string
	TypesEn   map[string]string
	Items     map[string]*GItem
	ByID      map[int]*GItem
	ByName    map[string]*GItem // nome inglese o italiano normalizzato
	Aspects   map[int][2]string // id affisso → [it, en]
	Skills    map[string][2]string
	Types     map[string]string // tipo oggetto → nome italiano
	Classes   []string          // indice classe → slug (barbarian…)
	ClassesIt map[string]string
	SetsIt    map[string]string
	Boards    map[string]PBoardDef
	Glyphs    map[string][2]string
	NodeNames map[string][2]string
}

// tavola paragon: larghezza e tipo di ogni casella (0 vuota, 1 normale, 2 magica, 3 rara, 4 leggendaria, 5 porta, 6 glifo, 7 inizio)
type PBoardDef struct {
	Width int
	Cells []byte
	Names []string
	It    string
	En    string
}

var gd struct {
	sync.RWMutex
	d     *gameData
	state string
}

func getGD() *gameData {
	gd.RLock()
	defer gd.RUnlock()
	return gd.d
}

var tagRe = regexp.MustCompile(`\[[a-z]{2}\]`)
var fmtRe = regexp.MustCompile(`\{[^}]*\}`)

func cleanName(s string) string {
	if i := strings.Index(s, "|"); i > 0 {
		s = s[:i]
	}
	s = tagRe.ReplaceAllString(s, "")
	s = fmtRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func normName(s string) string {
	s = strings.ToLower(cleanName(s))
	r := strings.NewReplacer("’", "'", "`", "'", "‘", "'", ",", "", ".", "", "-", " ", "  ", " ")
	return strings.TrimSpace(r.Replace(s))
}

func cacheFile(name string) string { return filepath.Join(dataDir, "cache", name) }

// scarica un file della CDN se manca o se è più vecchio di maxAge (usa If-Modified-Since)
func cdnFile(name string, maxAge time.Duration) (string, error) {
	p := cacheFile(name)
	st, err := os.Stat(p)
	if err == nil && time.Since(st.ModTime()) < maxAge {
		return p, nil
	}
	req, _ := http.NewRequest("GET", mxCDN+name, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Hellrise/"+appVersion)
	req.Header.Set("Accept-Encoding", "gzip")
	if err == nil {
		req.Header.Set("If-Modified-Since", st.ModTime().UTC().Format(http.TimeFormat))
	}
	resp, rerr := (&http.Client{Timeout: 3 * time.Minute}).Do(req)
	if rerr != nil {
		if err == nil {
			return p, nil // offline: va bene la copia che c'è
		}
		return "", rerr
	}
	defer resp.Body.Close()
	if resp.StatusCode == 304 {
		now := time.Now()
		os.Chtimes(p, now, now)
		return p, nil
	}
	if resp.StatusCode != 200 {
		if err == nil {
			return p, nil
		}
		return "", fmt.Errorf("%s: HTTP %d", name, resp.StatusCode)
	}
	var r io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, gerr := gzip.NewReader(resp.Body)
		if gerr != nil {
			return "", gerr
		}
		defer gz.Close()
		r = gz
	}
	tmp := p + ".tmp"
	f, ferr := os.Create(tmp)
	if ferr != nil {
		return "", ferr
	}
	_, cerr := io.Copy(f, r)
	f.Close()
	if cerr != nil {
		os.Remove(tmp)
		return "", cerr
	}
	os.Rename(tmp, p)
	return p, nil
}

type rawMin struct {
	Version string `json:"version"`
	Items   map[string]struct {
		ID          int      `json:"id"`
		Type        string   `json:"type"`
		MagicType   int      `json:"magicType"`
		ClassFilter []bool   `json:"classFilter"`
		Families    []string `json:"families"`
		Flags       int      `json:"flags"`
	} `json:"items"`
	Affixes map[string]struct {
		ID    int    `json:"id"`
		Power string `json:"power"`
	} `json:"affixes"`
	Classes map[string]struct {
		Name string `json:"name"`
	} `json:"classes"`
	ParagonBoards map[string]struct {
		Width int       `json:"width"`
		Nodes []*string `json:"nodes"`
	} `json:"paragonBoards"`
	ParagonNodes map[string]struct {
		Rarity int `json:"rarity"`
	} `json:"paragonNodes"`
}

type rawName = map[string]struct {
	Name string `json:"name"`
}

type rawLoc struct {
	ParagonBoards rawName `json:"paragonBoards"`
	ParagonGlyphs rawName `json:"paragonGlyphs"`
	ParagonNodes  rawName `json:"paragonNodes"`
	Items         map[string]struct {
		Name string `json:"name"`
	} `json:"items"`
	Affixes map[string]struct {
		Prefix string `json:"prefix"`
		Suffix string `json:"suffix"`
	} `json:"affixes"`
	Skills map[string]struct {
		Name string `json:"name"`
	} `json:"skills"`
	ItemTypes map[string]struct {
		Name string `json:"name"`
	} `json:"itemTypes"`
	Classes map[string]struct {
		NameMale string `json:"nameMale"`
	} `json:"classes"`
	ItemSets map[string]struct {
		Name string `json:"name"`
	} `json:"itemSets"`
}

func loadGameData() error {
	gd.Lock()
	gd.state = "Scarico i dati di gioco da Maxroll…"
	gd.Unlock()
	files := map[string]string{}
	for _, n := range []string{"data.min.json", "data.itit.json", "data.enus.json"} {
		p, err := cdnFile(n, 24*time.Hour)
		if err != nil {
			gd.Lock()
			gd.state = "Dati di gioco non disponibili: " + err.Error()
			gd.Unlock()
			return err
		}
		files[n] = p
	}
	var mn rawMin
	var it, en rawLoc
	if err := readJSON(files["data.min.json"], &mn); err != nil {
		return err
	}
	if err := readJSON(files["data.itit.json"], &it); err != nil {
		return err
	}
	if err := readJSON(files["data.enus.json"], &en); err != nil {
		return err
	}
	d := &gameData{Version: mn.Version, Items: map[string]*GItem{}, ByID: map[int]*GItem{}, ByName: map[string]*GItem{},
		Aspects: map[int][2]string{}, Skills: map[string][2]string{}, Types: map[string]string{}, ClassesIt: map[string]string{}, SetsIt: map[string]string{}}
	d.Classes = make([]string, 8)
	for k, c := range mn.Classes {
		var i int
		fmt.Sscanf(k, "%d", &i)
		if i >= 0 && i < 16 {
			for len(d.Classes) <= i {
				d.Classes = append(d.Classes, "")
			}
			d.Classes[i] = strings.ToLower(c.Name)
			d.ClassesIt[strings.ToLower(c.Name)] = it.Classes[k].NameMale
		}
	}
	d.TypesEn = map[string]string{}
	for k, t := range it.ItemTypes {
		d.Types[k] = cleanName(t.Name)
	}
	for k, t := range en.ItemTypes {
		d.TypesEn[k] = cleanName(t.Name)
	}
	for k, s := range it.ItemSets {
		d.SetsIt[k] = cleanName(s.Name)
	}
	for k, x := range mn.Items {
		g := &GItem{Key: k, ID: x.ID, Type: x.Type, Magic: x.MagicType, It: cleanName(it.Items[k].Name), En: cleanName(en.Items[k].Name)}
		if g.En == "" && g.It == "" {
			continue
		}
		all := true
		for _, b := range x.ClassFilter {
			all = all && b
		}
		if !all {
			for i, b := range x.ClassFilter {
				if b && i < len(d.Classes) {
					g.Classes = append(g.Classes, d.Classes[i])
				}
			}
		}
		for _, f := range x.Families {
			if strings.Contains(strings.ToLower(f), "mythic") {
				g.Mythic = true
			}
		}
		d.Items[k] = g
		d.ByID[x.ID] = g
	}
	// nomi → oggetto: prima i leggendari/unici "veri" (chiave più corta = versione base)
	keys := make([]string, 0, len(d.Items))
	for k := range d.Items {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := d.Items[keys[i]], d.Items[keys[j]]
		if a.Magic != b.Magic {
			return a.Magic > b.Magic
		}
		return len(keys[i]) < len(keys[j])
	})
	for _, k := range keys {
		g := d.Items[k]
		for _, n := range []string{g.En, g.It} {
			if nn := normName(n); nn != "" {
				if _, ok := d.ByName[nn]; !ok {
					d.ByName[nn] = g
				}
			}
		}
	}
	for _, a := range mn.Affixes {
		if a.Power == "" {
			continue
		}
		ai, ae := it.Affixes[a.Power], en.Affixes[a.Power]
		pi, pe := cleanName(ai.Prefix), cleanName(ae.Prefix)
		if pi == "" {
			pi = cleanName(ai.Suffix)
		}
		var nameEn string
		if s := cleanName(ae.Suffix); s != "" {
			nameEn = "Aspect " + s
		} else if pe != "" {
			nameEn = pe + " Aspect"
		}
		if pi == "" && nameEn == "" {
			continue
		}
		nameIt := ""
		if pi != "" {
			nameIt = "Aspetto " + pi
		}
		d.Aspects[a.ID] = [2]string{nameIt, nameEn}
	}
	for k, s := range it.Skills {
		d.Skills[k] = [2]string{cleanName(s.Name), cleanName(en.Skills[k].Name)}
	}
	d.Boards = map[string]PBoardDef{}
	for k, b := range mn.ParagonBoards {
		if b.Width <= 0 {
			continue
		}
		def := PBoardDef{Width: b.Width, Cells: make([]byte, len(b.Nodes)), Names: make([]string, len(b.Nodes)), It: cleanName(it.ParagonBoards[k].Name), En: cleanName(en.ParagonBoards[k].Name)}
		for i, n := range b.Nodes {
			if n == nil || *n == "" {
				continue
			}
			name := *n
			def.Names[i] = name
			switch {
			case strings.Contains(name, "Gate"):
				def.Cells[i] = 5
			case strings.Contains(name, "Socket"):
				def.Cells[i] = 6
			case strings.Contains(name, "Start"):
				def.Cells[i] = 7
			case strings.Contains(name, "_Legendary"):
				def.Cells[i] = 4
			case strings.Contains(name, "_Rare"):
				def.Cells[i] = 3
			case strings.Contains(name, "_Magic"):
				def.Cells[i] = 2
			default:
				def.Cells[i] = 1
				if r := mn.ParagonNodes[name].Rarity; r >= 3 {
					def.Cells[i] = 4
				}
			}
		}
		d.Boards[k] = def
	}
	d.Glyphs = map[string][2]string{}
	for k, g := range it.ParagonGlyphs {
		d.Glyphs[k] = [2]string{cleanName(g.Name), cleanName(en.ParagonGlyphs[k].Name)}
	}
	d.NodeNames = map[string][2]string{}
	for k, g := range it.ParagonNodes {
		d.NodeNames[k] = [2]string{cleanName(g.Name), cleanName(en.ParagonNodes[k].Name)}
	}
	gd.Lock()
	gd.d, gd.state = d, ""
	gd.Unlock()
	log.Printf("dati di gioco Maxroll %s: %d oggetti, %d aspetti, %d abilità", d.Version, len(d.Items), len(d.Aspects), len(d.Skills))
	invalidateSeason(0)
	go reloadOverlayParagon() // il paragon dell'overlay va riletto con i dati nuovi
	return nil
}

func gameDataLoop() {
	for {
		if err := loadGameData(); err != nil {
			log.Printf("dati di gioco: %v", err)
			time.Sleep(5 * time.Minute)
			continue
		}
		time.Sleep(24 * time.Hour)
	}
}

func gameDataState() string {
	gd.RLock()
	defer gd.RUnlock()
	return gd.state
}

// nome italiano di un oggetto citato in inglese (tabelle loot, guide)
func itName(en string) (string, *GItem) {
	d := getGD()
	if d == nil {
		return en, nil
	}
	if g := d.ByName[normName(en)]; g != nil && g.It != "" {
		return g.It, g
	}
	return en, nil
}

func aspectName(nid int) (string, string) {
	d := getGD()
	if d == nil {
		return "", ""
	}
	a := d.Aspects[nid]
	if a[0] == "" {
		a[0] = a[1]
	}
	return a[0], a[1]
}

func skillName(key string) string {
	if d := getGD(); d != nil {
		if s, ok := d.Skills[key]; ok && s[0] != "" {
			return s[0]
		}
	}
	// Barbarian_LungingStrike → Lunging Strike
	if i := strings.Index(key, "_"); i >= 0 {
		key = key[i+1:]
	}
	return regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(key, "$1 $2")
}

func classesIt() map[string]string {
	if d := getGD(); d != nil {
		return d.ClassesIt
	}
	return nil
}
