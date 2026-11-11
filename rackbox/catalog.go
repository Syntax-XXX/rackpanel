package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed apps.json
var builtinApps []byte

type appAsk struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Secret bool   `json:"secret,omitempty"`
}

type app struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Image  string            `json:"image"`
	Icon   string            `json:"icon,omitempty"`
	Desc   string            `json:"desc"`
	Web    string            `json:"web,omitempty"`
	Memory string            `json:"memory,omitempty"`
	Binds  []string          `json:"binds,omitempty"`
	Env    map[string]string `json:"env,omitempty"`
	Ask    []appAsk          `json:"ask,omitempty"`
	Setup  string            `json:"setup,omitempty"`
	Cmd    []string          `json:"cmd,omitempty"`
	Note   string            `json:"note,omitempty"`
}

// loadCatalog: built-in apps plus /etc/rackbox/apps.json (same format, same id replaces)
func loadCatalog() []app {
	var list []app
	_ = json.Unmarshal(builtinApps, &list)
	var extra []app
	if b, err := os.ReadFile("/etc/rackbox/apps.json"); err == nil && json.Unmarshal(b, &extra) == nil {
		for _, e := range extra {
			replaced := false
			for i := range list {
				if list[i].ID == e.ID {
					list[i], replaced = e, true
				}
			}
			if !replaced {
				list = append(list, e)
			}
		}
	}
	return list
}

// dataDir: where app data lives (a data volume when there is one)
func dataDir(id string) string {
	if fi, err := os.Stat("/volume1"); err == nil && fi.IsDir() {
		return filepath.Join("/volume1/apps", id)
	}
	return filepath.Join("/srv/apps", id)
}

func hostIP() string {
	if c, err := net.Dial("udp", "1.1.1.1:53"); err == nil {
		defer c.Close()
		return c.LocalAddr().(*net.UDPAddr).IP.String()
	}
	h, _ := os.Hostname()
	return h
}

var keyRe = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func installApp(id string, kv []string) error {
	var a *app
	for _, x := range loadCatalog() {
		if x.ID == id {
			x := x
			a = &x
		}
	}
	if a == nil {
		return fmt.Errorf("unknown app %q (see rackbox apps)", id)
	}
	given := map[string]string{}
	for _, s := range kv {
		k, v, ok := strings.Cut(s, "=")
		if !ok || !keyRe.MatchString(k) {
			return fmt.Errorf("expected KEY=VALUE, got %q", s)
		}
		given[k] = v
	}
	r := strings.NewReplacer("{data}", dataDir(a.ID), "{host}", hostIP())
	c := container{Name: a.ID, Image: a.Image, Memory: a.Memory, Web: a.Web, App: a.ID, Cmd: a.Cmd, Env: map[string]string{}}
	for k, v := range a.Env {
		c.Env[k] = r.Replace(v)
	}
	for _, q := range a.Ask {
		v := given[q.Key]
		if v == "" {
			return fmt.Errorf("%s (%s) is required", q.Label, q.Key)
		}
		c.Env[q.Key] = v
	}
	for _, b := range a.Binds {
		c.Binds = append(c.Binds, r.Replace(b))
	}
	if err := create(c, a.Setup); err != nil {
		return err
	}
	if err := systemctl("enable", "--now", unitName(c.Name)); err != nil {
		return err
	}
	if a.Note != "" {
		fmt.Println(r.Replace(a.Note))
	}
	if a.Web != "" {
		fmt.Printf("%s is starting: http://%s%s\n", a.Name, hostIP(), a.Web)
	} else {
		fmt.Printf("%s is running.\n", a.Name)
	}
	return nil
}
