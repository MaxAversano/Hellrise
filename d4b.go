package main

// Build di D4Builds: la pagina pubblica, letta come la vede un visitatore (con Edge nascosto),
// da cui si prendono equipaggiamento, abilità e rune, poi tradotti con i dati di gioco.

import (
	"crypto/sha1"
	"encoding/hex"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	d4bNameRe  = regexp.MustCompile(`(?s)class="builder__gear__name\s*(builder__gear__name--(\w+))?\s*"[^>]*>(.*?)</div>`)
	d4bSlotRe  = regexp.MustCompile(`(?s)class="builder__gear__slot"[^>]*>(.*?)</div>`)
	d4bGemRe   = regexp.MustCompile(`(?s)class="builder__gem__icon"[^>]*>\s*<img[^>]*alt="([^"]*)"`)
	d4bSkillRe = regexp.MustCompile(`(?s)class="build__skill__wrapper ([A-Za-z0-9]+)\s*"[^>]*>\s*<img[^>]*alt="([^"]*)"`)
	d4bRuneRe  = regexp.MustCompile(`(?s)class="build__skill__name build__skill__name--rune[^"]*"[^>]*>(.*?)<`)
	d4bVarRe   = regexp.MustCompile(`class="builder__variant__input"[^>]*value="([^"]*)"`)
	d4bMu      sync.Mutex
)

var slotIt = map[string]string{"Helm": "Elmo", "Chest Armor": "Torso", "Gloves": "Guanti", "Pants": "Pantaloni", "Boots": "Stivali",
	"Amulet": "Amuleto", "Ring 1": "Anello 1", "Ring 2": "Anello 2", "Weapon": "Arma", "Offhand": "Mano secondaria",
	"Bludgeoning Weapon": "Arma contundente", "Slashing Weapon": "Arma da taglio", "Dual-Wield Weapon 1": "Arma (mano 1)",
	"Dual-Wield Weapon 2": "Arma (mano 2)", "Ranged Weapon": "Arma a distanza", "Focus": "Focus", "Totem": "Totem", "Shield": "Scudo"}

func d4bBuildDetail(u string) (*PBuild, error) {
	if !strings.HasPrefix(u, "https://d4builds.gg/builds/") {
		return nil, &httpErr{400}
	}
	p := filepath.Join(dataDir, "cache", "d4b_"+hashURL(u)+".html")
	var page string
	if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 6*time.Hour {
		b, _ := os.ReadFile(p)
		page = string(b)
	}
	if page == "" {
		d4bMu.Lock()
		got, err := browserDump(u)
		d4bMu.Unlock()
		if err != nil || !strings.Contains(got, "builder__gear") {
			if b, e2 := os.ReadFile(p); e2 == nil {
				page = string(b)
			} else if err != nil {
				return nil, err
			} else {
				return nil, &httpErr{422}
			}
		} else {
			page = got
			os.WriteFile(p, []byte(got), 0o644)
		}
	}
	gdd := getGD()
	enSkill := map[string][2]string{} // nome inglese senza spazi → [it, en]
	aspIt := map[string]string{}
	if gdd != nil {
		for _, s := range gdd.Skills {
			if s[1] != "" {
				enSkill[squash(s[1])] = s
			}
		}
		for _, a := range gdd.Aspects {
			if a[1] != "" && a[0] != "" {
				aspIt[normName(a[1])] = a[0]
			}
		}
	}
	v := PVariant{Name: "Base"}
	if m := d4bVarRe.FindStringSubmatch(page); m != nil && m[1] != "" {
		v.Name = html.UnescapeString(m[1])
	}
	chunks := strings.Split(page, `class="builder__gear__item `)
	for _, ch := range chunks[1:] {
		if i := strings.Index(ch, `class="builder__gear__item `); i > 0 {
			ch = ch[:i]
		}
		m := d4bNameRe.FindStringSubmatch(ch)
		sm := d4bSlotRe.FindStringSubmatch(ch)
		if m == nil || sm == nil {
			continue
		}
		en := strings.TrimSpace(html.UnescapeString(tagStripRe.ReplaceAllString(m[3], "")))
		slot := strings.TrimSpace(html.UnescapeString(tagStripRe.ReplaceAllString(sm[1], "")))
		if en == "" {
			continue
		}
		var gems []string
		for _, gm := range d4bGemRe.FindAllStringSubmatch(ch, -1) {
			gname := html.UnescapeString(gm[1])
			if it, gi := itName(gname); gi != nil {
				gname = it + "|" + gi.En
			}
			gems = append(gems, gname)
		}
		g := PGear{En: en, Name: en, TypeEn: slot, Type: slotIt[slot], Kind: "unique", Sockets: gems}
		if g.Type == "" {
			g.Type = slot
		}
		switch m[2] {
		case "mythic":
			g.Kind = "mythic"
		}
		if strings.Contains(strings.ToLower(en), "aspect") {
			g.Kind = "legendary"
			if it := aspIt[normName(en)]; it != "" {
				g.Name = it
			} else if it := aspIt[normName(strings.TrimSuffix(strings.TrimPrefix(en, "Aspect of "), " Aspect"))]; it != "" {
				g.Name = it
			}
		} else if it, gi := itName(en); gi != nil {
			g.Name = it
			if gi.Mythic {
				g.Kind = "mythic"
			}
			if strings.HasPrefix(gi.Key, "Runeword_") {
				g.Kind = "runeword"
			}
		}
		v.Gear = append(v.Gear, g)
	}
	seen := map[string]bool{}
	for _, m := range d4bSkillRe.FindAllStringSubmatch(page, -1) {
		k, alt := m[1], html.UnescapeString(m[2])
		if seen[k] || len(v.Skills) >= 6 {
			continue
		}
		seen[k] = true
		s := PSkill{Key: k, Name: alt, En: alt}
		if x, ok := enSkill[squash(alt)]; ok {
			s.Name, s.En = x[0], x[1]
		} else if x, ok := enSkill[squash(k)]; ok {
			s.Name, s.En = x[0], x[1]
		} else if gdd != nil {
			continue // non è un'abilità (es. arma della tecnica)
		}
		v.Skills = append(v.Skills, s)
	}
	var runes []string
	for _, m := range d4bRuneRe.FindAllStringSubmatch(page, -1) {
		r := strings.TrimSpace(html.UnescapeString(m[1]))
		if r != "" && len(runes) < 8 {
			if it, gi := itName(r); gi != nil {
				r = it + "|" + gi.En
			}
			runes = append(runes, r)
		}
	}
	if len(runes) > 0 {
		v.Gear = append(v.Gear, PGear{Type: "Rune attive", TypeEn: "Active Runes", Name: "Rune", En: "Runes", Kind: "other", Sockets: runes})
	}
	if len(v.Gear) == 0 && len(v.Skills) == 0 {
		return nil, &httpErr{422}
	}
	return &PBuild{Name: v.Name, URL: u, Variants: []PVariant{v}}, nil
}

func squash(s string) string {
	return strings.ToLower(strings.NewReplacer(" ", "", "'", "", "’", "", "-", "", ":", "").Replace(s))
}

func hashURL(u string) string {
	sum := sha1.Sum([]byte(u))
	return hex.EncodeToString(sum[:8])
}
