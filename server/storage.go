package main

import (
	"bufio"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type raidInfo struct {
	Name    string   `json:"name"`
	Level   string   `json:"level"`
	State   string   `json:"state"`
	Members []string `json:"members"`
	Status  string   `json:"status"` // z. B. [4/3] [UUU_]
	Mount   string   `json:"mount"`
	Sync    string   `json:"sync,omitempty"`
}

var mdLine = regexp.MustCompile(`^(md\d+)\s*:\s*(\S+)\s+(?:\(\S+\)\s+)?(\S+)\s+(.*)$`)

func mdstat() []raidInfo {
	mounts := map[string]string{}
	for _, v := range volumes() {
		mounts[strings.TrimPrefix(v.Dev, "/dev/")] = v.Path
	}
	if b, err := os.ReadFile("/proc/swaps"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			p := strings.Fields(l)
			if len(p) > 0 && strings.HasPrefix(p[0], "/dev/md") {
				mounts[strings.TrimPrefix(p[0], "/dev/")] = "swap"
			}
		}
	}
	out := []raidInfo{}
	f, err := os.Open("/proc/mdstat")
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var cur *raidInfo
	for sc.Scan() {
		l := sc.Text()
		if m := mdLine.FindStringSubmatch(l); m != nil {
			out = append(out, raidInfo{Name: m[1], State: m[2], Level: m[3], Mount: mounts[m[1]]})
			cur = &out[len(out)-1]
			for _, d := range strings.Fields(m[4]) {
				cur.Members = append(cur.Members, regexp.MustCompile(`\[\d+\].*`).ReplaceAllString(d, ""))
			}
			continue
		}
		if cur != nil {
			t := strings.TrimSpace(l)
			if i := strings.Index(t, "["); i >= 0 && strings.Contains(t, "blocks") {
				cur.Status = strings.TrimSpace(t[i:])
			}
			if strings.Contains(t, "recovery") || strings.Contains(t, "resync") {
				cur.Sync = t
			}
		}
	}
	return out
}

var devRe = regexp.MustCompile(`^(sd[a-z]+|nvme\d+n\d+|hd[a-z]|vd[a-z]+|mmcblk\d+)$`)

func registerStorage(mux *http.ServeMux, s *storeT) {
	mux.HandleFunc("GET /api/storage", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		ok(w, map[string]any{"disks": readDisks(), "raid": mdstat(), "volumes": volumes()})
	}))
	mux.HandleFunc("GET /api/storage/smart/{dev}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		dev := r.PathValue("dev")
		if !devRe.MatchString(dev) {
			fail(w, 400, 400, "ungültig")
			return
		}
		b, _ := exec.Command("smartctl", "-n", "standby", "-a", "/dev/"+dev).CombinedOutput()
		ok(w, string(b))
	}))
	// Platte sofort schlafen legen bzw. Ruhezeit setzen (hdparm -S: 0=aus, 1..240 = n×5 s, 241..251 = (n-240)×30 min)
	mux.HandleFunc("POST /api/storage/{dev}/sleep", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		dev := r.PathValue("dev")
		if !devRe.MatchString(dev) || !strings.HasPrefix(dev, "sd") {
			fail(w, 400, 400, "ungültig")
			return
		}
		out, err := run("hdparm", "-y", "/dev/"+dev)
		if err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		disksCache.Lock()
		disksCache.at = time.Time{}
		disksCache.Unlock()
		ok(w, "schläft")
	}))
	mux.HandleFunc("PUT /api/storage/spindown", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in struct {
			Minutes int `json:"minutes"`
		}
		if readJSON(r, &in) != nil || in.Minutes < 0 || in.Minutes > 330 {
			fail(w, 400, 400, "0–330 Minuten")
			return
		}
		v := 0
		switch {
		case in.Minutes == 0:
		case in.Minutes <= 20:
			v = in.Minutes * 12
		default:
			v = 240 + (in.Minutes+29)/30
		}
		var errs []string
		for _, d := range readDisks() {
			if out, err := run("hdparm", "-S", strconv.Itoa(v), d.Path); err != nil {
				errs = append(errs, d.Name+": "+strings.TrimSpace(out))
			}
		}
		_ = os.WriteFile(*dataDir+"/spindown", []byte(strconv.Itoa(v)), 0o644)
		if len(errs) > 0 {
			fail(w, 200, 500, strings.Join(errs, "; "))
			return
		}
		ok(w, v)
	}))
	mux.HandleFunc("GET /api/storage/spindown", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		b, _ := os.ReadFile(*dataDir + "/spindown")
		v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		m := 0
		switch {
		case v == 0:
		case v <= 240:
			m = v / 12
		default:
			m = (v - 240) * 30
		}
		ok(w, m)
	}))
}

// applySpindown: gespeicherte Ruhezeit beim Start wieder setzen (hdparm-Werte gehen beim Neustart verloren)
func applySpindown() {
	b, err := os.ReadFile(*dataDir + "/spindown")
	if err != nil {
		return
	}
	v := strings.TrimSpace(string(b))
	for _, d := range readDisks() {
		_, _ = run("hdparm", "-S", v, d.Path)
	}
}
