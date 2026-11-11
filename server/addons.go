package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Add-ons: /usr/share/rackpanel/addons/<id>/addon.json + page.js (and any other static files).
// An add-on may call one program with an allow-list of first arguments; "read" commands also work
// with read-only API keys, "jobs" run in the background with live output.
//
//	{"id":"rackbox","title":"Containers","icon":"layers","version":"1.0","description":"…","page":"page.js",
//	 "exec":["/usr/bin/rackbox"],"read":["ls","images","apps","logs"],"write":["start","stop"],"jobs":["install","pull"]}
var addonsDir = flag.String("addons", "/usr/share/rackpanel/addons", "add-on directory")

type addon struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Icon        string   `json:"icon"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Page        string   `json:"page"`
	Exec        []string `json:"exec,omitempty"`
	Read        []string `json:"read,omitempty"`
	Write       []string `json:"write,omitempty"`
	Jobs        []string `json:"jobs,omitempty"`
}

var addonID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

func loadAddons() []addon {
	out := []addon{}
	ents, _ := os.ReadDir(*addonsDir)
	for _, e := range ents {
		var a addon
		b, err := os.ReadFile(filepath.Join(*addonsDir, e.Name(), "addon.json"))
		if err != nil || json.Unmarshal(b, &a) != nil || a.ID != e.Name() || !addonID.MatchString(a.ID) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func findAddon(id string) *addon {
	for _, a := range loadAddons() {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

func registerAddons(mux *http.ServeMux, s *storeT) {
	mux.HandleFunc("GET /api/addons", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		list := loadAddons()
		for i := range list {
			list[i].Exec = nil // the UI doesn't need program paths
		}
		ok(w, list)
	}))

	// static files of an add-on (no directory listings)
	files := http.StripPrefix("/addons/", http.FileServer(http.Dir(*addonsDir)))
	mux.HandleFunc("GET /addons/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || strings.Contains(r.URL.Path, "/.") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	})

	run := func(w http.ResponseWriter, a *addon, args []string, allowJob bool) {
		if len(a.Exec) == 0 || len(args) == 0 {
			fail(w, 400, 400, "this add-on has no commands")
			return
		}
		for _, x := range args {
			if strings.ContainsRune(x, 0) || len(x) > 4096 {
				fail(w, 400, 400, "invalid argument")
				return
			}
		}
		full := append(append([]string{}, a.Exec[1:]...), args...)
		if allowJob && slices.Contains(a.Jobs, args[0]) {
			j := startJob(a.Title+": "+strings.Join(args, " "), a.Exec[0], full...)
			ok(w, map[string]any{"job": j.ID})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		out, err := exec.CommandContext(ctx, a.Exec[0], full...).CombinedOutput()
		if err != nil {
			fail(w, 200, 500, strings.TrimSpace(lastLines(string(out), 8)))
			return
		}
		ok(w, map[string]any{"output": string(out)})
	}
	// GET: read commands only (also with read-only API keys)
	mux.HandleFunc("GET /api/addons/{id}/run", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		a := findAddon(r.PathValue("id"))
		args := r.URL.Query()["a"]
		if a == nil || len(args) == 0 || !slices.Contains(a.Read, args[0]) {
			fail(w, 403, 403, "not allowed")
			return
		}
		run(w, a, args, false)
	}))
	mux.HandleFunc("POST /api/addons/{id}/run", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		a := findAddon(r.PathValue("id"))
		var in struct {
			Args []string `json:"args"`
		}
		if a == nil || readJSON(r, &in) != nil || len(in.Args) == 0 ||
			!(slices.Contains(a.Read, in.Args[0]) || slices.Contains(a.Write, in.Args[0]) || slices.Contains(a.Jobs, in.Args[0])) {
			fail(w, 403, 403, "not allowed")
			return
		}
		run(w, a, in.Args, true)
	}))
}
