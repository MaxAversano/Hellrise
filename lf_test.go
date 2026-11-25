package main

import (
	"os"
	"strings"
	"testing"
)

func TestLootFilter(t *testing.T) {
	if _, err := os.Stat("/tmp/hrd/cache/data.min.json"); err != nil {
		t.Skip("dati di gioco non presenti")
	}
	dataDir = "/tmp/hrd"
	if err := loadGameData(); err != nil {
		t.Fatal(err)
	}
	q := LFReq{Name: "Flay Barb", Uniques: []string{"Tuskhelm of Joritz the Mighty", "Rage of Harrogath", "Blood-Mad Idol", "Arioc's Needle"}, Sets: []string{"Berserker's Crucible", "Bloodletter's Flow"}}
	q.Slots = append(q.Slots, struct {
		Slot  string   `json:"slot"`
		Stats []string `json:"stats"`
	}{"Boots", []string{"Willpower", "Maximum Life", "Movement Speed", "Profane Sentinel", "Movement Speed (Natural Motion - Mobility)"}}, struct {
		Slot  string   `json:"slot"`
		Stats []string `json:"stats"`
	}{"Ring 1", []string{"Critical Strike Damage Multiplier", "All Damage Multiplier", "Vulnerable Damage Multiplier", "Critical Strike Chance", "Ranks to All Skills"}})
	out := buildLootFilter(q)
	t.Logf("%+v", out)
	os.WriteFile("/tmp/lf/gen.txt", []byte(out.Code), 0o644)
}

func TestBoardNames(t *testing.T) {
	if _, err := os.Stat("/tmp/hrd/cache/data.min.json"); err != nil {
		t.Skip()
	}
	dataDir = "/tmp/hrd"
	loadGameData()
	d := getGD()
	for k, b := range d.Boards {
		if strings.HasSuffix(k, "_00") || strings.Contains(k, "Start") {
			t.Log(k, b.En, b.Width)
		}
	}
	for k, g := range d.Glyphs {
		if g[1] == "Marshal" || g[1] == "Executioner" {
			t.Log(k, g)
		}
	}
}
