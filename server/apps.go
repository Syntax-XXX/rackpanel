package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Apps: native Debian-Pakete mit Dienst und (optional) Weboberfläche. Der Katalog liegt als apps.json
// neben der Oberfläche und kommt mit dem Panel-Paket – so laufen Apps auch auf ARMv5 mit 512 MB,
// wo es für Container weder Images noch genug Speicher gibt.
type app struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Desc     string   `json:"desc"`
	Icon     string   `json:"icon"`
	Packages []string `json:"packages"`
	Service  string   `json:"service"`
	Port     int      `json:"port"`
	Path     string   `json:"path"`
	Note     string   `json:"note"`
	Setup    string   `json:"setup,omitempty"` // Shell, läuft nach der Installation als root (Katalog gehört root)
}

var pkgRe = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)

func loadApps() []app {
	list := []app{}
	b, err := os.ReadFile(filepath.Join(*wwwDir, "apps.json"))
	if err != nil || json.Unmarshal(b, &list) != nil {
		return []app{}
	}
	return list
}

func findApp(id string) *app {
	for _, a := range loadApps() {
		if a.ID == id && len(a.Packages) > 0 {
			for _, p := range a.Packages {
				if !pkgRe.MatchString(p) {
					return nil
				}
			}
			return &a
		}
	}
	return nil
}

func registerApps(mux *http.ServeMux, s *storeT) {
	mux.HandleFunc("GET /api/apps", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		out := []map[string]any{}
		for _, a := range loadApps() {
			if len(a.Packages) == 0 {
				continue
			}
			st, _ := run("dpkg-query", "-W", "-f=${db:Status-Abbrev}", a.Packages[0])
			m := map[string]any{"id": a.ID, "name": a.Name, "desc": a.Desc, "icon": a.Icon, "packages": a.Packages,
				"service": a.Service, "port": a.Port, "path": a.Path, "note": a.Note,
				"installed": strings.HasPrefix(st, "ii"), "active": false}
			if a.Service != "" && m["installed"] == true {
				act, _ := run("systemctl", "is-active", a.Service)
				m["active"] = strings.TrimSpace(act) == "active"
			}
			out = append(out, m)
		}
		ok(w, out)
	}))
	mux.HandleFunc("POST /api/apps/{id}/{action}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		a := findApp(r.PathValue("id"))
		if a == nil {
			fail(w, 404, 404, "App unbekannt")
			return
		}
		switch r.PathValue("action") {
		case "install":
			script := `set -e; apt-get update -q; apt-get install -y "$@"` + "\n" + a.Setup
			ok(w, startJob(a.Name+" installieren", "sh", append([]string{"-c", script, "sh"}, a.Packages...)...).ID)
		case "remove": // Einstellungen bleiben liegen (kein purge)
			ok(w, startJob(a.Name+" entfernen", "apt-get", append([]string{"remove", "-y"}, a.Packages...)...).ID)
		default:
			fail(w, 400, 400, "ungültig")
		}
	}))
}
