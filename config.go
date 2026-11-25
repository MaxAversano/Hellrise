package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Settings struct {
	RemoteURL      string       `json:"remoteUrl"`   // server dati (es. https://www.aversano.design/hr-data/)
	RemoteToken    string       `json:"remoteToken"` // codice segreto del server (lo stesso di BetterStomp)
	Season         int          `json:"season"`      // 0 = l'ultima disponibile
	Class          string       `json:"class"`       // ultima classe scelta
	NativeFrame    bool         `json:"nativeFrame"`
	Autostart      bool         `json:"autostart"`
	KeepInTray     bool         `json:"keepInTray"` // resta nella barra per le notifiche anche a finestra chiusa
	Notifications  bool         `json:"notifications"`
	NotifyBoss     bool         `json:"notifyBoss"`
	NotifyHelltide bool         `json:"notifyHelltide"`
	NotifyLegion   bool         `json:"notifyLegion"`
	NotifyBefore   int          `json:"notifyBefore"` // minuti prima
	D4PressurePath string       `json:"d4pressurePath"`
	Favorites      []string     `json:"favorites"`   // oggetti seguiti in "Chi droppa cosa"
	SavedBuilds    []SavedBuild `json:"savedBuilds"` // build salvate (con filtri e variante)
	Blur           bool         `json:"blur"`        // effetto vetro più marcato
	Motion         bool         `json:"motion"`      // animazioni dello sfondo
	Accent         string       `json:"accent"`      // colore principale
	Lang           string       `json:"lang"`        // nomi di gioco: "it" tradotti, "en" originali
	Overlay        OverlaySet   `json:"overlay"`
}

type OWidget struct {
	On   bool    `json:"on"`
	Lock bool    `json:"lock"` // bloccato: non si sposta
	X    float64 `json:"x"`    // 0..1 frazione dello schermo
	Y    float64 `json:"y"`
}

// overlay in gioco (Diablo IV in finestra senza bordi)
type OverlaySet struct {
	Enabled   bool           `json:"enabled"`
	Always    bool           `json:"always"` // mostralo anche fuori dal gioco (per provarlo)
	Scale     float64        `json:"scale"`
	Timers    OWidget        `json:"timers"`
	TBoss     bool           `json:"tBoss"`
	THelltide bool           `json:"tHelltide"`
	TLegion   bool           `json:"tLegion"`
	Paragon   OWidget        `json:"paragon"`
	Planner   string         `json:"planner"` // planner Maxroll della build scelta
	Title     string         `json:"title"`
	Variant   int            `json:"variant"`
	Step      int            `json:"step"`
	Board     int            `json:"board"`
	Cell      int            `json:"cell"`   // lato di una casella in pixel (a 1080p), riquadro
	Mode      string         `json:"mode"`   // "panel" riquadro con la tavola | "align" sovrapposto alla tavola del gioco
	CellPx    float64        `json:"cellPx"` // passo delle caselle sullo schermo in pixel (modalità sovrapposta)
	AlignX    float64        `json:"alignX"` // angolo in alto a sinistra della griglia (frazione dello schermo)
	AlignY    float64        `json:"alignY"`
	Rot       map[string]int `json:"rot"`      // rotazione scelta da te per ogni tavola (quarti di giro)
	Opacity   int            `json:"opacity"`  // 20-100 %
	MoveKey   int            `json:"moveKey"`  // tasto da tenere premuto per spostare e bloccare i riquadri (codice virtuale Windows)
	Rotation  OWidget        `json:"rotation"` // riquadro con la rotazione delle abilità
	RotTitle  string         `json:"rotTitle"`
	RotLines  []string       `json:"rotLines"`
}

func defaultSettings() Settings {
	return Settings{
		RemoteURL:     "https://www.aversano.design/hr-data/",
		Notifications: true, NotifyBoss: true, NotifyHelltide: false, NotifyLegion: false, NotifyBefore: 5,
		Blur: true, Motion: true, Accent: "ember", Lang: "it",
		Overlay: OverlaySet{Enabled: true, Scale: 1, Timers: OWidget{On: true, X: 0.012, Y: 0.28}, TBoss: true, THelltide: true, TLegion: true,
			Paragon: OWidget{On: true, X: 0.69, Y: 0.14}, Rotation: OWidget{On: false, X: 0.012, Y: 0.55}, Cell: 13, Mode: "auto", Opacity: 92, MoveKey: 0xA4, CellPx: 40, AlignX: 0.3, AlignY: 0.2},
	}
}

var (
	setMu    sync.RWMutex
	settings = defaultSettings()
)

func settingsPath() string { return filepath.Join(dataDir, "settings.json") }

func loadSettings() {
	s := defaultSettings()
	readJSON(settingsPath(), &s)
	if s.RemoteToken == "" {
		s.RemoteToken = betterStompToken()
	}
	if s.Overlay.Opacity <= 0 {
		s.Overlay.Opacity = 92
	}
	if s.Overlay.MoveKey == 0 {
		s.Overlay.MoveKey = 0xA4
	}
	if s.Lang == "" {
		s.Lang = "it"
	}
	if s.NotifyBefore <= 0 {
		s.NotifyBefore = 5
	}
	setMu.Lock()
	settings = s
	setMu.Unlock()
	saveSettings()
}

func getSettings() Settings {
	setMu.RLock()
	defer setMu.RUnlock()
	return settings
}

func updateSettings(fn func(*Settings)) Settings {
	setMu.Lock()
	fn(&settings)
	s := settings
	setMu.Unlock()
	saveSettings()
	return s
}

func saveSettings() { writeJSON(settingsPath(), getSettings()) }

// il server di Hellrise usa lo stesso codice di BetterStomp: se BetterStomp è sul Desktop lo prende da lì
func betterStompToken() string {
	exe, _ := os.Executable()
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(filepath.Dir(filepath.Dir(exe)), "BetterStomp", "BetterStomp-data", "settings.json"),
		filepath.Join(home, "Desktop", "BetterStomp", "BetterStomp-data", "settings.json"),
	} {
		var bs struct {
			RemoteToken string `json:"remoteToken"`
			RemoteURL   string `json:"remoteUrl"`
		}
		if readJSON(p, &bs) == nil && bs.RemoteToken != "" && strings.Contains(bs.RemoteURL, "aversano.design") {
			return bs.RemoteToken
		}
	}
	return ""
}

type SavedBuild struct {
	URL     string   `json:"url"`
	Src     string   `json:"src"`
	Title   string   `json:"title"`
	Cls     string   `json:"cls"`
	Type    string   `json:"type"`
	Season  int      `json:"season"`
	Profile string   `json:"profile,omitempty"`
	Variant int      `json:"variant"`
	Tab     string   `json:"tab,omitempty"`
	Note    string   `json:"note,omitempty"`
	At      int64    `json:"at"`
	Seen    int64    `json:"seen"`            // data di aggiornamento della build vista l'ultima volta
	Told    int64    `json:"told"`            // data per cui è già partito l'avviso di Windows
	Owned   []string `json:"owned,omitempty"` // pezzi che hai già ("Helm", "Ring 1"…)
}
