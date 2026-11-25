package main

// Mobalytics blocca il server (HTTP 403): build in evidenza e guida dei boss si scaricano dal PC,
// ogni 3 ore, e si uniscono ai dati della stagione.

import (
	"encoding/json"
	"html"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type mobaBuild struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Cls      string   `json:"cls"`
	Season   int      `json:"season"`
	Type     string   `json:"type"`
	Activity []string `json:"activity"`
	Author   string   `json:"author"`
	URL      string   `json:"url"`
	Verified bool     `json:"verified"`
	Likes    int      `json:"likes"`
	Updated  int64    `json:"updated"`
}

type mobaBoss struct {
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
}

type mobaCache struct {
	At     int64       `json:"at"`
	Builds []mobaBuild `json:"builds"`
	Bosses []mobaBoss  `json:"bosses"`
}

var moba struct {
	sync.Mutex
	c mobaCache
}

func mobaGet(u string) (string, error) {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := (&http.Client{Timeout: 40 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", &httpErr{resp.StatusCode}
	}
	return string(b), err
}

type httpErr struct{ code int }

func (e *httpErr) Error() string { return "HTTP " + itoa(e.code) }

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

var seasonNumRe = regexp.MustCompile(`(\d+)`)

func mobaDocs(page string) []mobaBuild {
	i := strings.Index(page, "window.__PRELOADED_STATE__=")
	if i < 0 {
		return nil
	}
	i += len("window.__PRELOADED_STATE__=")
	end := strings.Index(page[i:], "</script>")
	if end < 0 {
		return nil
	}
	raw := strings.TrimSuffix(strings.TrimSpace(page[i:i+end]), ";")
	var st any
	if json.Unmarshal([]byte(raw), &st) != nil {
		return nil
	}
	var out []mobaBuild
	var walk func(o any, d int)
	walk = func(o any, d int) {
		if d > 30 {
			return
		}
		switch v := o.(type) {
		case map[string]any:
			if _, ok := v["slugifiedName"]; ok {
				if data, ok := v["data"].(map[string]any); ok {
					if b, ok := mobaDoc(v, data); ok {
						out = append(out, b)
					}
					return
				}
			}
			for _, x := range v {
				walk(x, d+1)
			}
		case []any:
			for _, x := range v {
				walk(x, d+1)
			}
		}
	}
	walk(st, 0)
	return out
}

func mobaDoc(v, data map[string]any) (mobaBuild, bool) {
	str := func(m map[string]any, k string) string { s, _ := m[k].(string); return s }
	tags := map[string][]string{}
	if t, ok := v["tags"].(map[string]any); ok {
		if arr, ok := t["data"].([]any); ok {
			for _, x := range arr {
				if m, ok := x.(map[string]any); ok {
					tags[str(m, "groupSlug")] = append(tags[str(m, "groupSlug")], str(m, "slug"))
				}
			}
		}
	}
	if len(tags["class"]) == 0 {
		return mobaBuild{}, false
	}
	b := mobaBuild{ID: str(v, "id"), Name: html.UnescapeString(str(data, "name")), Cls: tags["class"][0], Type: "Endgame", Activity: tags["activity"]}
	for _, s := range tags["season"] {
		if m := seasonNumRe.FindString(s); m != "" {
			var n int
			json.Unmarshal([]byte(m), &n)
			if n > b.Season {
				b.Season = n
			}
		}
	}
	for _, t := range tags["build-type"] {
		if t == "leveling" {
			b.Type = "Leveling"
		}
	}
	for _, t := range tags["verified"] {
		b.Verified = b.Verified || t == "verified"
	}
	author, _ := v["author"].(map[string]any)
	if author != nil {
		b.Author = str(author, "name")
	}
	if f, ok := v["featured"].(map[string]any); ok && str(f, "slug") != "" {
		b.URL = "https://mobalytics.gg/diablo-4/builds/" + str(f, "slug")
	} else {
		short := ""
		if author != nil {
			short = str(author, "shortName")
			if u, ok := author["user"].(map[string]any); ok && short == "" {
				short = str(u, "username")
			}
		}
		b.URL = "https://mobalytics.gg/diablo-4/profile/" + short + "/builds/" + str(v, "slugifiedName")
	}
	if f, ok := v["favorites"].(map[string]any); ok {
		if n, ok := f["favoritesCount"].(float64); ok {
			b.Likes = int(n)
		}
	}
	if t, err := time.Parse(time.RFC3339, str(v, "updatedAt")); err == nil {
		b.Updated = t.Unix()
	}
	return b, b.Name != ""
}

var blockRe = regexp.MustCompile(`(?s)<(h2|p|li)\b[^>]*>(.*?)</(h2|p|li)>`)
var tagStripRe = regexp.MustCompile(`(?s)<[^>]+>`)

func mobaBossGuide(page string) []mobaBoss {
	var out []mobaBoss
	cur := -1
	done := false
	for _, m := range blockRe.FindAllStringSubmatch(page, -1) {
		txt := strings.Join(strings.Fields(html.UnescapeString(tagStripRe.ReplaceAllString(m[2], " "))), " ")
		if txt == "" {
			continue
		}
		if m[1] == "h2" {
			low := strings.ToLower(txt)
			if strings.Contains(low, "table of contents") || strings.Contains(low, "see more") || strings.Contains(low, "introduction") {
				cur = -1
				continue
			}
			out = append(out, mobaBoss{Name: txt})
			cur, done = len(out)-1, false
			continue
		}
		if cur < 0 || done {
			continue
		}
		low := strings.ToLower(txt)
		if low == "non-class specific" || low == "barbarian" || low == "weapons" || low == "equipment" || low == "jewellery" || low == "jewelry" {
			done = true
			continue
		}
		if len(out[cur].Lines) < 8 && len(txt) < 600 {
			out[cur].Lines = append(out[cur].Lines, txt)
		}
	}
	res := out[:0]
	for _, b := range out {
		if len(b.Lines) > 0 {
			res = append(res, b)
		}
	}
	return res
}

func mobaRefresh() {
	pages := []string{"builds"}
	for _, c := range []string{"barbarian", "druid", "necromancer", "rogue", "sorcerer", "spiritborn", "paladin", "warlock"} {
		pages = append(pages, c+"-builds")
	}
	seen := map[string]bool{}
	var builds []mobaBuild
	fails := 0
	for _, p := range pages {
		page, err := mobaGet("https://mobalytics.gg/diablo-4/" + p)
		if err != nil {
			fails++
			log.Printf("mobalytics: %s: %v", p, err)
			continue
		}
		for _, b := range mobaDocs(page) {
			if !seen[b.URL] {
				seen[b.URL] = true
				builds = append(builds, b)
			}
		}
		time.Sleep(1500 * time.Millisecond)
	}
	var bosses []mobaBoss
	if page, err := mobaGet("https://mobalytics.gg/diablo-4/guides/diablo-4-boss-loot-tables"); err == nil {
		bosses = mobaBossGuide(page)
	} else {
		log.Printf("mobalytics: guida boss: %v", err)
	}
	moba.Lock()
	if len(builds) > 0 {
		moba.c.Builds = builds
	}
	if len(bosses) > 0 {
		moba.c.Bosses = bosses
	}
	if len(builds) > 0 || len(bosses) > 0 {
		moba.c.At = time.Now().Unix()
	}
	c := moba.c
	moba.Unlock()
	writeJSON(cacheFile("mobalytics.json"), c)
	log.Printf("mobalytics: %d build, %d boss (%d pagine non riuscite)", len(builds), len(bosses), fails)
	invalidateSeason(0)
}

func mobaLoop() {
	var c mobaCache
	readJSON(cacheFile("mobalytics.json"), &c)
	moba.Lock()
	moba.c = c
	moba.Unlock()
	time.Sleep(45 * time.Second)
	for {
		moba.Lock()
		age := time.Since(time.Unix(moba.c.At, 0))
		moba.Unlock()
		if age > 3*time.Hour {
			mobaRefresh()
		}
		time.Sleep(10 * time.Minute)
	}
}

func mobaSnapshot() mobaCache {
	moba.Lock()
	defer moba.Unlock()
	return moba.c
}
