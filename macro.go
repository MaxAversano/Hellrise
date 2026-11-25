package main

// Profili macro suggeriti per D4Pressure: per ogni build propone tasto, modalità e ritardo di ogni
// abilità da impostare a mano in D4Pressure. Hellrise non crea file per D4Pressure e non invia mai tasti al gioco.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type MacroRow struct {
	Skill   string `json:"skill"`   // nome abilità (etichetta)
	Key     string `json:"key"`     // "LMB", "RMB", "1"…"4", "Q", "Space"…
	Mode    string `json:"mode"`    // Tap | Hold
	DelayMs int    `json:"delayMs"` // per Tap
	On      bool   `json:"on"`      // attiva nel profilo D4Pressure
}

type MacroProfile struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`  // nome del personaggio/profilo
	Class   string     `json:"class"` // Barbarian, Necromancer…
	Build   string     `json:"build"` // titolo della build
	Planner string     `json:"planner,omitempty"`
	Notes   string     `json:"notes"`
	Rows    []MacroRow `json:"rows"`
	Updated int64      `json:"updated"`
}

var macroMu sync.Mutex

func macroPath() string { return filepath.Join(dataDir, "macro.json") }

func loadMacros() []MacroProfile {
	var m []MacroProfile
	readJSON(macroPath(), &m)
	return m
}

func saveMacros(m []MacroProfile) error { return writeJSON(macroPath(), m) }

var d4pCache struct {
	sync.Mutex
	at  time.Time
	run bool
}

// controllo leggero (al massimo ogni 15 secondi) se D4Pressure è aperto
func d4pressureRunning() bool {
	d4pCache.Lock()
	defer d4pCache.Unlock()
	if time.Since(d4pCache.at) < 15*time.Second {
		return d4pCache.run
	}
	out, err := hiddenCmd("tasklist", "/FI", "IMAGENAME eq D4Pressure.exe", "/NH").Output()
	d4pCache.run = err == nil && strings.Contains(strings.ToLower(string(out)), "d4pressure.exe")
	d4pCache.at = time.Now()
	return d4pCache.run
}

func findD4Pressure() string {
	if p := getSettings().D4PressurePath; p != "" && fileExists(p) {
		return p
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, "Desktop", "D4Pressure", "D4Pressure.exe"),
		filepath.Join(home, "Desktop", "D4Pressure.exe"),
		filepath.Join(home, "Downloads", "D4Pressure", "D4Pressure.exe"),
		filepath.Join(home, "Downloads", "D4Pressure.exe"),
		filepath.Join(home, "Desktop", "Utility", "D4Pressure", "D4Pressure.exe"),
		filepath.Join(home, "Desktop", "Utility", "D4Pressure.exe"),
	} {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func launchD4Pressure() error {
	p := findD4Pressure()
	if p == "" {
		return fmt.Errorf("D4Pressure.exe non trovato: indica il percorso nelle impostazioni")
	}
	c := exec.Command(p)
	c.Dir = filepath.Dir(p)
	return c.Start()
}

var d4Cache struct {
	sync.Mutex
	at  time.Time
	run bool
}

// Diablo IV è aperto? (controllo leggero, al massimo ogni 15 secondi)
func diabloRunning() bool {
	d4Cache.Lock()
	defer d4Cache.Unlock()
	if time.Since(d4Cache.at) < 15*time.Second {
		return d4Cache.run
	}
	out, err := hiddenCmd("tasklist", "/FI", "IMAGENAME eq Diablo IV.exe", "/NH").Output()
	d4Cache.run = err == nil && strings.Contains(strings.ToLower(string(out)), "diablo iv.exe")
	d4Cache.at = time.Now()
	return d4Cache.run
}
