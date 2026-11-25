package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func setupAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/whoami", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, map[string]string{"app": "Hellrise"})
	})

	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		jsonOut(w, map[string]any{
			"version":   appVersion,
			"settings":  getSettings(),
			"seasons":   seasonsAvailable(),
			"season":    currentSeason(atoi(r.URL.Query().Get("s"))),
			"remote":    remoteStatus(),
			"gameData":  gameDataState(),
			"frameless": isFrameless(),
			"update":    updReady.Load(),
			"d4p":       map[string]any{"path": findD4Pressure(), "running": d4pressureRunning()},
			"classesIt": classesIt(),
			"diablo":    diabloRunning(),
			"collector": collectorInfo(),
		})
	})

	mux.HandleFunc("/api/collect", func(w http.ResponseWriter, r *http.Request) {
		if !isCollector() {
			jsonErr(w, 400, "la raccolta si fa solo dal PC con la cartella del server")
			return
		}
		go collect()
		w.WriteHeader(204)
	})

	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) {
		err := remoteRefresh()
		if err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		jsonOut(w, remoteStatus())
	})

	// build di una classe in una stagione (Maxroll, D4Builds, Mobalytics + tier list di Rob)
	mux.HandleFunc("/api/builds", func(w http.ResponseWriter, r *http.Request) {
		n := currentSeason(atoi(r.URL.Query().Get("s")))
		v := seasonView(n)
		if v == nil {
			jsonErr(w, 404, "Nessun dato per questa stagione: controlla il server nelle impostazioni")
			return
		}
		cls := r.URL.Query().Get("cls")
		var list []BuildEntry
		for _, b := range v.Builds {
			if cls == "" || b.Cls == cls {
				list = append(list, b)
			}
		}
		jsonOut(w, map[string]any{"season": n, "builds": list, "rob": v.Rob, "updated": v.Updated})
	})

	mux.HandleFunc("/api/mobabuild", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if b := serverDetail("moba", u); b != nil {
			jsonOut(w, b)
			return
		}
		if !isCollector() {
			jsonErr(w, 404, "build non ancora raccolta: arriverà con il prossimo aggiornamento dei dati")
			return
		}
		b, err := mobaBuildDetail(u)
		if err != nil {
			jsonErr(w, 502, "Mobalytics: "+err.Error())
			return
		}
		jsonOut(w, b)
	})

	mux.HandleFunc("/api/d4bbuild", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if b := serverDetail("d4b", u); b != nil {
			jsonOut(w, b)
			return
		}
		if !isCollector() {
			jsonErr(w, 404, "build non ancora raccolta: arriverà con il prossimo aggiornamento dei dati")
			return
		}
		b, err := d4bBuildDetail(u)
		if err != nil {
			jsonErr(w, 502, "D4Builds: "+err.Error())
			return
		}
		jsonOut(w, b)
	})

	// build D4Builds completa (tutte le schede), già pronta sul server
	mux.HandleFunc("/api/d4bfull", func(w http.ResponseWriter, r *http.Request) {
		u := keyURL(r.URL.Query().Get("u"))
		if !strings.HasPrefix(u, "https://d4builds.gg/builds/") {
			jsonErr(w, 400, "indirizzo non valido")
			return
		}
		u += "/"
		if b := serverFull(u); b != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write(b)
			return
		}
		if !isCollector() {
			jsonErr(w, 404, "questa build arriverà con il prossimo aggiornamento dei dati")
			return
		}
		f, err := d4bFullRead(u)
		if err != nil {
			jsonErr(w, 502, "D4Builds: "+err.Error())
			return
		}
		go uploadFull(u, f)
		jsonOut(w, f)
	})

	// paragon usato dall'overlay (planner Maxroll o build D4Builds)
	mux.HandleFunc("/api/ovparagon", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var b *PBuild
		var err error
		if strings.HasPrefix(id, "d4b:") {
			b, err = d4bParagonBuild(strings.TrimPrefix(id, "d4b:"))
		} else {
			b, err = plannerBuild(id)
		}
		if err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		jsonOut(w, b)
	})

	// filtro loot del gioco creato dalla build
	mux.HandleFunc("/api/lootfilter", func(w http.ResponseWriter, r *http.Request) {
		var q LFReq
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		out := buildLootFilter(q)
		if out.Code == "" {
			jsonErr(w, 503, "dati di gioco non ancora pronti")
			return
		}
		jsonOut(w, out)
	})

	mux.HandleFunc("/api/guide", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if strings.HasPrefix(u, "https://") {
			go openGuideWindow(u)
		}
		w.WriteHeader(204)
	})

	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		b, err := plannerBuild(r.URL.Query().Get("id"))
		if err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		jsonOut(w, b)
	})

	// chi droppa cosa
	mux.HandleFunc("/api/loot", func(w http.ResponseWriter, r *http.Request) {
		n := currentSeason(atoi(r.URL.Query().Get("s")))
		v := seasonView(n)
		if v == nil {
			jsonErr(w, 404, "Nessun dato per questa stagione")
			return
		}
		jsonOut(w, map[string]any{"season": n, "items": v.Items, "bosses": v.Bosses, "sources": v.LootSrc, "ptr": v.PTR, "names": getGD() != nil})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, eventsInfo())
	})

	mux.HandleFunc("/api/map", func(w http.ResponseWriter, r *http.Request) {
		b, err := mapMarkers()
		if err != nil {
			jsonErr(w, 502, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	})
	mux.HandleFunc("/api/tile/", serveTile)
	mux.HandleFunc("/api/img", serveImg)
	mux.HandleFunc("/api/mapfound", func(w http.ResponseWriter, r *http.Request) {
		found := map[string]bool{}
		readJSON(mapFoundPath(), &found)
		if r.Method == "POST" {
			var in map[string]bool
			if json.NewDecoder(r.Body).Decode(&in) == nil {
				for k, v := range in {
					if v {
						found[k] = true
					} else {
						delete(found, k)
					}
				}
				if r.URL.Query().Get("reset") == "1" {
					found = map[string]bool{}
				}
				writeJSON(mapFoundPath(), found)
			}
		}
		jsonOut(w, found)
	})

	mux.HandleFunc("/api/patches", func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(serverCache("patches.json"))
		if err != nil {
			jsonErr(w, 404, "Note delle patch non ancora scaricate dal server")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	})

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			jsonOut(w, getSettings())
			return
		}
		var in map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		old := getSettings()
		s := updateSettings(func(s *Settings) {
			b, _ := json.Marshal(s)
			cur := map[string]json.RawMessage{}
			json.Unmarshal(b, &cur)
			for k, v := range in {
				cur[k] = v
			}
			b, _ = json.Marshal(cur)
			json.Unmarshal(b, s)
		})
		if s.Overlay.Planner != old.Overlay.Planner {
			reloadOverlayParagon()
		}
		if s.Autostart != old.Autostart {
			go setAutostart(s.Autostart)
		}
		if s.RemoteURL != old.RemoteURL || s.RemoteToken != old.RemoteToken {
			go remoteRefresh()
		}
		jsonOut(w, s)
	})

	// profili macro / D4Pressure
	mux.HandleFunc("/api/macro", func(w http.ResponseWriter, r *http.Request) {
		macroMu.Lock()
		defer macroMu.Unlock()
		list := loadMacros()
		if r.Method == "GET" {
			if list == nil {
				list = []MacroProfile{}
			}
			jsonOut(w, list)
			return
		}
		var p MacroProfile
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			jsonErr(w, 400, err.Error())
			return
		}
		if r.URL.Query().Get("delete") == "1" {
			out := list[:0]
			for _, x := range list {
				if x.ID != p.ID {
					out = append(out, x)
				}
			}
			saveMacros(out)
			jsonOut(w, out)
			return
		}
		if p.ID == "" {
			p.ID = strconv.FormatInt(time.Now().UnixNano(), 36)
		}
		p.Updated = time.Now().Unix()
		found := false
		for i := range list {
			if list[i].ID == p.ID {
				list[i], found = p, true
			}
		}
		if !found {
			list = append(list, p)
		}
		saveMacros(list)
		jsonOut(w, p)
	})

	mux.HandleFunc("/api/macro/launch", func(w http.ResponseWriter, r *http.Request) {
		if err := launchD4Pressure(); err != nil {
			jsonErr(w, 500, err.Error())
			return
		}
		jsonOut(w, map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/open", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if strings.HasPrefix(u, "https://") {
			openURL(u)
		}
		w.WriteHeader(204)
	})

	mux.HandleFunc("/api/win", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch a := q.Get("a"); a {
		case "update":
			if err := selfUpdate(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(204)
			go restartApp()
		case "open":
			go openAppWindow(appURL)
			w.WriteHeader(204)
		case "inner":
			setInner(atof(q.Get("h")), atof(q.Get("d")))
			w.WriteHeader(204)
		case "quit":
			w.WriteHeader(204)
			go func() { time.Sleep(200 * time.Millisecond); appExit() }()
		default:
			windowAction(a)
			w.WriteHeader(204)
		}
	})
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }
func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
