package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ── Hintergrund-Jobs (apt, Neustart …) mit mitlaufender Ausgabe ──
type job struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Running  bool      `json:"running"`
	ExitCode int       `json:"exit_code"`
	Started  time.Time `json:"started"`
	out      bytes.Buffer
	mu       sync.Mutex
}

func (j *job) Write(b []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.out.Len() > 2<<20 { // höchstens 2 MB behalten
		j.out.Reset()
		j.out.WriteString("… (gekürzt)\n")
	}
	return j.out.Write(b)
}

var jobs = struct {
	sync.Mutex
	m map[string]*job
}{m: map[string]*job{}}

func startJob(title string, name string, args ...string) *job {
	id := make([]byte, 6)
	_, _ = rand.Read(id)
	j := &job{ID: hex.EncodeToString(id), Title: title, Running: true, Started: time.Now()}
	jobs.Lock()
	jobs.m[j.ID] = j
	jobs.Unlock()
	go func() {
		cmd := exec.Command(name, args...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive", "LANG=C.UTF-8")
		cmd.Stdout, cmd.Stderr = j, j
		err := cmd.Run()
		j.mu.Lock()
		j.Running = false
		if ee, ok := err.(*exec.ExitError); ok {
			j.ExitCode = ee.ExitCode()
		} else if err != nil {
			j.ExitCode = -1
			fmt.Fprintf(&j.out, "\nFehler: %v\n", err)
		}
		j.mu.Unlock()
	}()
	return j
}

var aptBusy sync.Mutex

func run(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).CombinedOutput()
	return string(b), err
}

var svcName = regexp.MustCompile(`^[a-zA-Z0-9@._-]+\.service$`)

// Dienste, die im Panel auftauchen (der Rest ist Systeminternes)
var listedServices = []string{"ssh.service", "nfs-server.service", "smbd.service", "nmbd.service", "rs812-fan.service", "rackpanel.service",
	"smartmontools.service", "systemd-timesyncd.service", "cron.service", "mdmonitor.service", "wsdd.service", "avahi-daemon.service", "unattended-upgrades.service"}

func registerAdmin(mux *http.ServeMux, s *storeT) {
	// Dienste
	mux.HandleFunc("GET /api/services", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		list := []map[string]any{}
		for _, n := range listedServices {
			out, _ := run("systemctl", "show", n, "--no-pager", "-p", "LoadState,ActiveState,SubState,UnitFileState,Description")
			m := map[string]any{"name": n}
			for _, l := range strings.Split(out, "\n") {
				if k, v, okk := strings.Cut(l, "="); okk {
					m[k] = v
				}
			}
			if m["LoadState"] == "not-found" {
				continue
			}
			list = append(list, m)
		}
		ok(w, list)
	}))
	mux.HandleFunc("POST /api/services/{name}/{action}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		n, a := r.PathValue("name"), r.PathValue("action")
		if !svcName.MatchString(n) || !map[string]bool{"start": true, "stop": true, "restart": true, "enable": true, "disable": true}[a] {
			fail(w, 400, 400, "ungültig")
			return
		}
		if n == "rackpanel.service" && (a == "stop" || a == "disable") {
			fail(w, 400, 400, "Das Panel kann sich nicht selbst abschalten")
			return
		}
		if n == "rackpanel.service" && a == "restart" {
			ok(w, "Neustart in 1 s")
			go func() { time.Sleep(time.Second); _, _ = run("systemctl", "restart", n) }()
			return
		}
		out, err := run("systemctl", a, n)
		if err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		ok(w, "ok")
	}))

	// NFS-Freigaben (/etc/exports) lesen und schreiben
	mux.HandleFunc("GET /api/shares/nfs", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		b, _ := os.ReadFile("/etc/exports")
		active, _ := run("exportfs", "-v")
		ok(w, map[string]any{"content": string(b), "active": active, "installed": have("exportfs")})
	}))
	mux.HandleFunc("PUT /api/shares/nfs", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		if readJSON(r, &in) != nil {
			fail(w, 400, 400, "ungültig")
			return
		}
		old, _ := os.ReadFile("/etc/exports")
		_ = os.WriteFile("/etc/exports.bak", old, 0o644)
		if err := os.WriteFile("/etc/exports", []byte(in["content"]), 0o644); err != nil {
			fail(w, 200, 500, err.Error())
			return
		}
		if out, err := run("exportfs", "-ra"); err != nil {
			_ = os.WriteFile("/etc/exports", old, 0o644) // bei Fehler alte Fassung zurück
			_, _ = run("exportfs", "-ra")
			fail(w, 200, 500, "Fehler, alte Freigaben wiederhergestellt: "+strings.TrimSpace(out))
			return
		}
		ok(w, "übernommen")
	}))

	// Updates: Liste der verfügbaren Pakete, dann apt upgrade als Job
	mux.HandleFunc("POST /api/updates/check", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		if !aptBusy.TryLock() {
			fail(w, 200, 409, "apt läuft bereits")
			return
		}
		defer aptBusy.Unlock()
		_, _ = run("apt-get", "update", "-qq")
		out, _ := run("apt", "list", "--upgradable")
		var pk []string
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "upgradable from") {
				pk = append(pk, l)
			}
		}
		ok(w, pk)
	}))
	mux.HandleFunc("POST /api/updates/install", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		j := startJob("Updates installieren", "sh", "-c", "apt-get update -q && apt-get -y -o Dpkg::Options::=--force-confold upgrade")
		ok(w, j.ID)
	}))
	mux.HandleFunc("POST /api/packages/install", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		_ = readJSON(r, &in)
		pkgs := strings.Fields(in["packages"])
		for _, p := range pkgs {
			if !regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`).MatchString(p) {
				fail(w, 400, 400, "ungültiger Paketname: "+p)
				return
			}
		}
		if len(pkgs) == 0 {
			fail(w, 400, 400, "keine Pakete")
			return
		}
		j := startJob("Installieren: "+strings.Join(pkgs, " "), "apt-get", append([]string{"install", "-y"}, pkgs...)...)
		ok(w, j.ID)
	}))
	mux.HandleFunc("GET /api/jobs/{id}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		jobs.Lock()
		j := jobs.m[r.PathValue("id")]
		jobs.Unlock()
		if j == nil {
			fail(w, 404, 404, "unbekannt")
			return
		}
		j.mu.Lock()
		defer j.mu.Unlock()
		ok(w, map[string]any{"id": j.ID, "title": j.Title, "running": j.Running, "exit_code": j.ExitCode, "output": j.out.String()})
	}))

	// Protokolle
	mux.HandleFunc("GET /api/logs", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		unit := r.URL.Query().Get("unit")
		args := []string{"-n", "400", "--no-pager", "-o", "short-iso"}
		if unit != "" {
			if !svcName.MatchString(unit) {
				fail(w, 400, 400, "ungültig")
				return
			}
			args = append(args, "-u", unit)
		}
		if r.URL.Query().Get("kernel") == "1" {
			args = append(args, "-k")
		}
		out, _ := run("journalctl", args...)
		ok(w, out)
	}))

	// Energie
	mux.HandleFunc("POST /api/power/{action}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		a := r.PathValue("action")
		if a != "reboot" && a != "poweroff" {
			fail(w, 400, 400, "ungültig")
			return
		}
		ok(w, a)
		go func() { time.Sleep(2 * time.Second); _, _ = run("systemctl", a) }()
	}))
}
