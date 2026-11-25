package main

// Build dettagliata da un planner Maxroll: varianti (leveling, starter, endgame…), barra delle abilità
// ed equipaggiamento, con i nomi italiani.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PGear struct {
	Slot     int      `json:"slot"`
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	En       string   `json:"en,omitempty"`
	Kind     string   `json:"kind"` // unique | mythic | runeword | legendary | talisman | other
	TypeEn   string   `json:"typeEn,omitempty"`
	Aspect   string   `json:"aspect,omitempty"`
	AspectEn string   `json:"aspectEn,omitempty"`
	Sockets  []string `json:"sockets,omitempty"`
	Tempered int      `json:"tempered,omitempty"`
}

type PSkill struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	En   string `json:"en,omitempty"`
}

type PVariant struct {
	Name    string   `json:"name"`
	Level   int      `json:"level"`
	Skills  []PSkill `json:"skills"`
	Gear    []PGear  `json:"gear"`
	Paragon []PStep  `json:"paragon,omitempty"`
}

type PBuild struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Class    string     `json:"class"`
	Season   string     `json:"season"`
	Updated  string     `json:"updated"`
	URL      string     `json:"url"`
	Variants []PVariant `json:"variants"`
	Filters  []PFilter  `json:"filters"`
}

type PFilter struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

var plannerID = regexp.MustCompile(`^[a-z0-9]{4,16}$`)

func plannerRaw(id string) ([]byte, error) {
	p := filepath.Join(dataDir, "planner", id+".json")
	if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 6*time.Hour {
		return os.ReadFile(p)
	}
	req, _ := http.NewRequest("GET", "https://planners.maxroll.gg/profiles/d4/"+id, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Hellrise/"+appVersion)
	resp, err := (&http.Client{Timeout: time.Minute}).Do(req)
	if err != nil {
		if b, e2 := os.ReadFile(p); e2 == nil {
			return b, nil
		}
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		if b2, e2 := os.ReadFile(p); e2 == nil {
			return b2, nil
		}
		return nil, fmt.Errorf("planner Maxroll: HTTP %d", resp.StatusCode)
	}
	os.WriteFile(p, b, 0o644)
	return b, nil
}

func plannerBuild(id string) (*PBuild, error) {
	if !plannerID.MatchString(id) {
		return nil, fmt.Errorf("planner non valido")
	}
	b, err := plannerRaw(id)
	if err != nil {
		return nil, err
	}
	var top struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Class  string `json:"class"`
		Data   string `json:"data"`
		Season any    `json:"season"`
		Date   string `json:"date"`
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	var data struct {
		Profiles []struct {
			Name     string         `json:"name"`
			Level    int            `json:"level"`
			Items    map[string]int `json:"items"`
			SkillBar []string       `json:"skillBar"`
			Paragon  struct {
				Steps []struct {
					Name string `json:"name"`
					Data struct {
						Boards []rawBoard `json:"boards"`
					} `json:"data"`
				} `json:"steps"`
			} `json:"paragon"`
		} `json:"profiles"`
		LootFilters []PFilter `json:"lootFilters"`
		Items       map[string]struct {
			ID       string              `json:"id"`
			Name     string              `json:"name"`
			Aspects  []struct{ Nid int } `json:"aspects"`
			Sockets  []string            `json:"sockets"`
			Tempered []json.RawMessage   `json:"tempered"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(top.Data), &data); err != nil {
		return nil, err
	}
	gdd := getGD()
	out := &PBuild{ID: id, Name: top.Name, Class: strings.ToLower(top.Class), Season: fmt.Sprint(top.Season), Updated: top.Date, URL: "https://maxroll.gg/d4/planner/" + id}
	for _, f := range data.LootFilters {
		if f.Code != "" {
			out.Filters = append(out.Filters, f)
		}
	}
	for _, p := range data.Profiles {
		v := PVariant{Name: p.Name, Level: p.Level}
		for _, sk := range p.SkillBar {
			s := PSkill{Key: sk, Name: skillName(sk)}
			if gdd != nil {
				s.En = gdd.Skills[sk][1]
			}
			v.Skills = append(v.Skills, s)
		}
		slots := make([]int, 0, len(p.Items))
		for k := range p.Items {
			n, _ := strconv.Atoi(k)
			slots = append(slots, n)
		}
		sort.Ints(slots)
		for _, sl := range slots {
			it, ok := data.Items[strconv.Itoa(p.Items[strconv.Itoa(sl)])]
			if !ok {
				continue
			}
			g := PGear{Slot: sl, Kind: "other", Tempered: len(it.Tempered)}
			var gi *GItem
			if gdd != nil {
				gi = gdd.Items[it.ID]
			}
			if gi != nil {
				g.Type, g.TypeEn = gdd.Types[gi.Type], gdd.TypesEn[gi.Type]
				g.Name, g.En = gi.It, gi.En
				switch {
				case strings.HasPrefix(it.ID, "Runeword_"):
					g.Kind = "runeword"
				case strings.HasPrefix(it.ID, "Talisman_"):
					g.Kind = "talisman"
					if strings.Contains(it.ID, "_Unique_") {
						g.Kind = "unique"
					}
				case gi.Mythic:
					g.Kind = "mythic"
				case strings.Contains(it.ID, "_Unique_"):
					g.Kind = "unique"
				case strings.Contains(it.ID, "_Legendary_"):
					g.Kind = "legendary"
				}
			}
			if g.Name == "" {
				g.Name = prettyKey(it.ID)
			}
			for _, a := range it.Aspects {
				if ni, ne := aspectName(a.Nid); ni != "" {
					g.Aspect, g.AspectEn = ni, ne
					break
				}
			}
			if g.Kind == "legendary" && g.Aspect != "" {
				g.Name, g.En = g.Aspect, g.AspectEn
				g.Aspect, g.AspectEn = "", ""
			}
			for _, s := range it.Sockets {
				name := prettyKey(s)
				if gdd != nil {
					if x := gdd.Items[s]; x != nil && x.It != "" {
						name = x.It + "|" + x.En
					}
				}
				g.Sockets = append(g.Sockets, name)
			}
			v.Gear = append(v.Gear, g)
		}
		for _, st := range p.Paragon.Steps {
			ps := PStep{Name: st.Name}
			for _, bd := range st.Data.Boards {
				ps.Boards = append(ps.Boards, paragonBoard(gdd, bd))
			}
			if len(ps.Boards) > 0 {
				v.Paragon = append(v.Paragon, ps)
			}
		}
		out.Variants = append(out.Variants, v)
	}
	return out, nil
}

var camelRe = regexp.MustCompile(`([a-z])([A-Z])`)

func prettyKey(k string) string {
	parts := strings.Split(k, "_")
	var keep []string
	for _, p := range parts {
		if p == "" || regexp.MustCompile(`^(\d+|S\d+|Generic|Unique|Legendary|Rune|Condition|Effect)$`).MatchString(p) {
			continue
		}
		keep = append(keep, camelRe.ReplaceAllString(p, "$1 $2"))
	}
	return strings.Join(keep, " ")
}
