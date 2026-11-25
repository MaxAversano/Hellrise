package main

// Timer degli eventi del mondo: boss del mondo (orari e zone dal server, presi da helltides.com),
// Maree infernali (ogni ora, 55 minuti) e Legioni (ogni 25 minuti), calcolate qui.

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type WBZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Boss string `json:"boss"`
}

type WorldBoss struct {
	T     int64    `json:"t"`
	Boss  string   `json:"boss"`
	Zones []WBZone `json:"zones"`
	Guess bool     `json:"guess,omitempty"` // orario stimato (oltre i dati del server)
}

type Event struct {
	Kind  string   `json:"kind"` // boss | helltide | legion
	Start int64    `json:"start"`
	End   int64    `json:"end"`
	Title string   `json:"title"`
	Zones []WBZone `json:"zones,omitempty"`
	Guess bool     `json:"guess,omitempty"`
}

var wb struct {
	sync.Mutex
	list   []WorldBoss
	at     int64
	source string
}

const (
	wbPeriod       = 210 * 60 // 3 ore e mezza
	legionPeriod   = 25 * 60
	legionDur      = 4 * 60
	helltidePeriod = 3600
	helltideDur    = 55 * 60
)

// riferimento delle Legioni (verificato il 25/11/2026: 11:55 UTC)
var legionAnchor = time.Date(2026, 11, 25, 11, 55, 0, 0, time.UTC).Unix()

func updateWorldBoss(raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var ev struct {
		At        int64       `json:"at"`
		Source    string      `json:"source"`
		WorldBoss []WorldBoss `json:"worldBoss"`
	}
	if json.Unmarshal(raw, &ev) != nil || len(ev.WorldBoss) == 0 {
		return
	}
	wb.Lock()
	wb.list, wb.at, wb.source = ev.WorldBoss, ev.At, ev.Source
	wb.Unlock()
}

func upcomingEvents(now time.Time, hours int) []Event {
	n := now.Unix()
	until := n + int64(hours)*3600
	var out []Event
	// boss del mondo
	wb.Lock()
	list := append([]WorldBoss(nil), wb.list...)
	wb.Unlock()
	var last int64
	for _, b := range list {
		last = b.T
		if b.T+15*60 < n || b.T > until {
			continue
		}
		out = append(out, Event{Kind: "boss", Start: b.T, End: b.T + 15*60, Title: b.Boss, Zones: b.Zones})
	}
	if last == 0 {
		// nessun dato dal server: riferimento verificato (Azmodan, 25/11/2026 13:00 UTC)
		last = time.Date(2026, 11, 25, 13, 0, 0, 0, time.UTC).Unix()
		for last+wbPeriod < n {
			last += wbPeriod
		}
		last -= wbPeriod
	}
	for t := last + wbPeriod; t <= until; t += wbPeriod {
		if t+15*60 < n {
			continue
		}
		out = append(out, Event{Kind: "boss", Start: t, End: t + 15*60, Title: "Boss del mondo", Guess: true})
	}
	// Maree infernali
	h0 := n - n%helltidePeriod
	for t := h0; t <= until; t += helltidePeriod {
		if t+helltideDur < n {
			continue
		}
		out = append(out, Event{Kind: "helltide", Start: t, End: t + helltideDur, Title: "Marea infernale"})
	}
	// Legioni
	k := (n - legionAnchor) / legionPeriod
	for t := legionAnchor + k*legionPeriod; t <= until; t += legionPeriod {
		if t+legionDur < n {
			continue
		}
		out = append(out, Event{Kind: "legion", Start: t, End: t + legionDur, Title: "Raduno della Legione"})
	}
	sortEvents(out)
	return out
}

func sortEvents(e []Event) {
	for i := 1; i < len(e); i++ {
		for j := i; j > 0 && e[j].Start < e[j-1].Start; j-- {
			e[j], e[j-1] = e[j-1], e[j]
		}
	}
}

func eventsInfo() map[string]any {
	wb.Lock()
	at, src := wb.at, wb.source
	wb.Unlock()
	return map[string]any{"now": time.Now().Unix(), "events": upcomingEvents(time.Now(), 24), "bossAt": at, "bossSource": src}
}

// notifiche di Windows prima degli eventi scelti
func eventNotifyLoop() {
	sent := map[string]bool{}
	for {
		time.Sleep(15 * time.Second)
		s := getSettings()
		if !s.Notifications {
			continue
		}
		now := time.Now()
		before := int64(s.NotifyBefore) * 60
		for _, e := range upcomingEvents(now, 2) {
			want := e.Kind == "boss" && s.NotifyBoss || e.Kind == "helltide" && s.NotifyHelltide || e.Kind == "legion" && s.NotifyLegion
			if !want {
				continue
			}
			left := e.Start - now.Unix()
			id := fmt.Sprintf("%s-%d", e.Kind, e.Start)
			if left > 0 && left <= before && !sent[id] {
				sent[id] = true
				where := ""
				if len(e.Zones) > 0 {
					where = " · " + e.Zones[0].Name
					for _, z := range e.Zones[1:] {
						where += ", " + z.Name
					}
				}
				notify(e.Title+" tra "+fmt.Sprint((left+59)/60)+" min", time.Unix(e.Start, 0).Format("15:04")+where)
			}
		}
		if len(sent) > 200 {
			sent = map[string]bool{}
		}
	}
}
