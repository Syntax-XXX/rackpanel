package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ── Systembenutzer (für SMB/NFS/SSH), Gruppen, SMB-Freigaben, Netzwerk, Hostname, Zeit ──

var (
	sysUserRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	hostRe    = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,62})$`)
	zoneRe    = regexp.MustCompile(`^[A-Za-z0-9_+/-]{1,64}$`)
)

// runIn: Befehl mit Eingabe auf stdin (Passwörter nie als Argument, sonst stehen sie in ps)
func runIn(stdin string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func have(bin string) bool { _, err := exec.LookPath(bin); return err == nil }

type sysUser struct {
	Name   string   `json:"name"`
	UID    int      `json:"uid"`
	Home   string   `json:"home"`
	Shell  string   `json:"shell"`
	Groups []string `json:"groups"`
	SMB    bool     `json:"smb"`
}

func smbUsers() map[string]bool {
	m := map[string]bool{}
	if !have("pdbedit") {
		return m
	}
	out, _ := run("pdbedit", "-L")
	for _, l := range strings.Split(out, "\n") {
		if n, _, f := strings.Cut(l, ":"); f {
			m[n] = true
		}
	}
	return m
}

func groupMembers() map[string][]string {
	m := map[string][]string{} // Benutzer → Gruppen
	b, _ := os.ReadFile("/etc/group")
	for _, l := range strings.Split(string(b), "\n") {
		p := strings.Split(l, ":")
		if len(p) < 4 || p[3] == "" {
			continue
		}
		for _, u := range strings.Split(p[3], ",") {
			m[u] = append(m[u], p[0])
		}
	}
	return m
}

func listUsers() []sysUser {
	out := []sysUser{}
	smb, gm := smbUsers(), groupMembers()
	b, _ := os.ReadFile("/etc/passwd")
	for _, l := range strings.Split(string(b), "\n") {
		p := strings.Split(l, ":")
		if len(p) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(p[2])
		if (uid < 1000 || uid >= 60000) && uid != 0 {
			continue
		}
		out = append(out, sysUser{p[0], uid, p[5], p[6], gm[p[0]], smb[p[0]]})
	}
	return out
}

func isNormalUser(name string) bool {
	for _, u := range listUsers() {
		if u.Name == name && u.UID >= 1000 {
			return true
		}
	}
	return false
}

func setSysPassword(name, pw string, smb bool) (string, error) {
	if out, err := runIn(name+":"+pw+"\n", "chpasswd"); err != nil {
		return out, err
	}
	if smb && have("smbpasswd") {
		return runIn(pw+"\n"+pw+"\n", "smbpasswd", "-a", "-s", name)
	}
	return "", nil
}

func registerSystem(mux *http.ServeMux, s *storeT) {
	// Benutzer
	mux.HandleFunc("GET /api/users", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var groups []string
		b, _ := os.ReadFile("/etc/group")
		for _, l := range strings.Split(string(b), "\n") {
			p := strings.Split(l, ":")
			if len(p) >= 3 {
				if gid, _ := strconv.Atoi(p[2]); gid >= 100 && gid < 60000 || p[0] == "sudo" {
					groups = append(groups, p[0])
				}
			}
		}
		ok(w, map[string]any{"users": listUsers(), "groups": groups, "samba": have("smbpasswd")})
	}))
	mux.HandleFunc("POST /api/users", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in struct {
			Name     string   `json:"name"`
			Password string   `json:"password"`
			Groups   []string `json:"groups"`
			SMB      bool     `json:"smb"`
			Shell    bool     `json:"shell"`
		}
		if readJSON(r, &in) != nil || !sysUserRe.MatchString(in.Name) || len(in.Password) < 5 {
			fail(w, 400, 400, "Name (klein, ohne Leerzeichen) und Passwort (mind. 5 Zeichen) nötig")
			return
		}
		shell := "/usr/sbin/nologin"
		if in.Shell {
			shell = "/bin/bash"
		}
		args := []string{"-m", "-s", shell}
		for _, g := range in.Groups {
			if !sysUserRe.MatchString(g) {
				fail(w, 400, 400, "ungültige Gruppe: "+g)
				return
			}
		}
		if len(in.Groups) > 0 {
			args = append(args, "-G", strings.Join(in.Groups, ","))
		}
		if out, err := run("useradd", append(args, in.Name)...); err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		if out, err := setSysPassword(in.Name, in.Password, in.SMB); err != nil {
			fail(w, 200, 500, "Benutzer angelegt, Passwort fehlgeschlagen: "+strings.TrimSpace(out))
			return
		}
		ok(w, in.Name)
	}))
	mux.HandleFunc("PUT /api/users/{name}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		n := r.PathValue("name")
		var in struct {
			Password string   `json:"password"`
			Groups   []string `json:"groups"`
			SMB      *bool    `json:"smb"`
		}
		if readJSON(r, &in) != nil || !isNormalUser(n) {
			fail(w, 400, 400, "unbekannter Benutzer")
			return
		}
		if in.Groups != nil {
			for _, g := range in.Groups {
				if !sysUserRe.MatchString(g) {
					fail(w, 400, 400, "ungültige Gruppe: "+g)
					return
				}
			}
			if out, err := run("usermod", "-G", strings.Join(in.Groups, ","), n); err != nil {
				fail(w, 200, 500, strings.TrimSpace(out))
				return
			}
		}
		smb := smbUsers()[n]
		if in.SMB != nil && !*in.SMB && smb {
			_, _ = run("smbpasswd", "-x", n)
			smb = false
		}
		if in.Password != "" {
			if len(in.Password) < 5 {
				fail(w, 400, 400, "Passwort zu kurz")
				return
			}
			if out, err := setSysPassword(n, in.Password, smb || in.SMB != nil && *in.SMB); err != nil {
				fail(w, 200, 500, strings.TrimSpace(out))
				return
			}
		} else if in.SMB != nil && *in.SMB && !smb {
			fail(w, 400, 400, "Für SMB-Zugang bitte ein Passwort setzen")
			return
		}
		ok(w, n)
	}))
	mux.HandleFunc("DELETE /api/users/{name}", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		n := r.PathValue("name")
		if !isNormalUser(n) {
			fail(w, 400, 400, "unbekannter Benutzer")
			return
		}
		if smbUsers()[n] {
			_, _ = run("smbpasswd", "-x", n)
		}
		// Home-Ordner bleibt bewusst erhalten (Daten)
		if out, err := run("userdel", n); err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		ok(w, n)
	}))
	mux.HandleFunc("POST /api/groups", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		if readJSON(r, &in) != nil || !sysUserRe.MatchString(in["name"]) {
			fail(w, 400, 400, "ungültiger Gruppenname")
			return
		}
		if out, err := run("groupadd", in["name"]); err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		ok(w, in["name"])
	}))

	// SMB-Freigaben: smb.conf mit Prüfung durch testparm, bei Fehler alte Fassung zurück
	mux.HandleFunc("GET /api/shares/smb", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		b, _ := os.ReadFile("/etc/samba/smb.conf")
		ok(w, map[string]any{"installed": have("smbd"), "content": string(b)})
	}))
	mux.HandleFunc("PUT /api/shares/smb", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		if readJSON(r, &in) != nil || !have("testparm") {
			fail(w, 400, 400, "Samba ist nicht installiert")
			return
		}
		old, _ := os.ReadFile("/etc/samba/smb.conf")
		_ = os.WriteFile("/etc/samba/smb.conf.bak", old, 0o644)
		if err := os.WriteFile("/etc/samba/smb.conf", []byte(in["content"]), 0o644); err != nil {
			fail(w, 200, 500, err.Error())
			return
		}
		if out, err := run("testparm", "-s", "--suppress-prompt"); err != nil {
			_ = os.WriteFile("/etc/samba/smb.conf", old, 0o644)
			fail(w, 200, 500, "Fehler, alte Fassung wiederhergestellt: "+lastLines(out, 6))
			return
		}
		_, _ = run("systemctl", "reload-or-restart", "smbd")
		ok(w, "übernommen")
	}))

	// Netzwerk (nur lesen; IP-Konfiguration bleibt bei /etc/network bzw. NetworkManager)
	mux.HandleFunc("GET /api/network", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		addr, _ := exec.Command("ip", "-j", "addr").Output()
		route, _ := exec.Command("ip", "-j", "route").Output()
		res, _ := os.ReadFile("/etc/resolv.conf")
		var dns []string
		for _, l := range strings.Split(string(res), "\n") {
			if v, f := strings.CutPrefix(strings.TrimSpace(l), "nameserver"); f {
				dns = append(dns, strings.TrimSpace(v))
			}
		}
		raw := func(b []byte) any {
			if json.Valid(b) && len(b) > 0 {
				return json.RawMessage(b)
			}
			return []any{}
		}
		speed := map[string]string{}
		ents, _ := os.ReadDir("/sys/class/net")
		for _, e := range ents {
			if b, err := os.ReadFile("/sys/class/net/" + e.Name() + "/speed"); err == nil {
				speed[e.Name()] = strings.TrimSpace(string(b))
			}
		}
		ok(w, map[string]any{"hostname": hostname(), "addr": raw(addr), "route": raw(route), "dns": dns, "speed": speed})
	}))
	mux.HandleFunc("PUT /api/hostname", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		if readJSON(r, &in) != nil || !hostRe.MatchString(in["hostname"]) {
			fail(w, 400, 400, "ungültiger Name (Buchstaben, Ziffern, Bindestrich)")
			return
		}
		h := in["hostname"]
		if out, err := run("hostnamectl", "set-hostname", h); err != nil {
			fail(w, 200, 500, strings.TrimSpace(out))
			return
		}
		// 127.0.1.1-Zeile mitziehen, sonst meckert sudo/Samba
		b, _ := os.ReadFile("/etc/hosts")
		lines := strings.Split(string(b), "\n")
		found := false
		for i, l := range lines {
			if strings.HasPrefix(l, "127.0.1.1") {
				lines[i], found = "127.0.1.1\t"+h, true
			}
		}
		if !found {
			lines = append([]string{"127.0.1.1\t" + h}, lines...)
		}
		_ = os.WriteFile("/etc/hosts", []byte(strings.Join(lines, "\n")), 0o644)
		ok(w, h)
	}))

	// Zeit
	mux.HandleFunc("GET /api/time", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		out, _ := run("timedatectl", "show")
		m := map[string]string{}
		for _, l := range strings.Split(out, "\n") {
			if k, v, f := strings.Cut(l, "="); f {
				m[k] = v
			}
		}
		zones, _ := run("timedatectl", "list-timezones")
		ok(w, map[string]any{"info": m, "zones": strings.Fields(zones)})
	}))
	mux.HandleFunc("PUT /api/time", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in struct {
			Zone string `json:"zone"`
			NTP  *bool  `json:"ntp"`
		}
		if readJSON(r, &in) != nil {
			fail(w, 400, 400, "ungültig")
			return
		}
		if in.Zone != "" {
			if !zoneRe.MatchString(in.Zone) {
				fail(w, 400, 400, "ungültige Zeitzone")
				return
			}
			if out, err := run("timedatectl", "set-timezone", in.Zone); err != nil {
				fail(w, 200, 500, strings.TrimSpace(out))
				return
			}
		}
		if in.NTP != nil {
			if out, err := run("timedatectl", "set-ntp", strconv.FormatBool(*in.NTP)); err != nil {
				fail(w, 200, 500, strings.TrimSpace(out))
				return
			}
		}
		ok(w, "ok")
	}))
}

func lastLines(s string, n int) string {
	l := strings.Split(strings.TrimSpace(s), "\n")
	if len(l) > n {
		l = l[len(l)-n:]
	}
	return strings.Join(l, " | ")
}
