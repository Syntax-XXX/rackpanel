// RackBox – runs Docker/OCI images without a daemon.
// Images are pulled straight from the registry (the right architecture, ARMv5 included), flattened into a plain
// directory and every container runs as its own systemd service via systemd-nspawn. Nothing stays resident
// besides the containers themselves, and neither overlayfs nor a container runtime is needed.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var version = "dev"

// image config (the parts that matter for running)
type imageConfig struct {
	Config struct {
		Env        []string `json:"Env"`
		Entrypoint []string `json:"Entrypoint"`
		Cmd        []string `json:"Cmd"`
		WorkingDir string   `json:"WorkingDir"`
		User       string   `json:"User"`
	} `json:"config"`
}

type imageMeta struct {
	Ref      string    `json:"ref"`
	Digest   string    `json:"digest"`
	Platform string    `json:"platform"`
	Size     int64     `json:"size"`
	Pulled   time.Time `json:"pulled"`
}

type container struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Digest  string            `json:"digest"`
	Env     map[string]string `json:"env,omitempty"`
	Binds   []string          `json:"binds,omitempty"`
	Cmd     []string          `json:"cmd,omitempty"`
	Memory  string            `json:"memory,omitempty"`
	Web     string            `json:"web,omitempty"` // e.g. ":8080/" – shown as a link in the panel
	App     string            `json:"app,omitempty"`
	Created time.Time         `json:"created"`
}

var root string

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "rackbox: "+f+"\n", a...)
	os.Exit(1)
}

// storeRoot: /etc/rackbox.conf "root=…", else the first data volume (the system disk is small on a NAS)
func storeRoot() string {
	if r := os.Getenv("RACKBOX_ROOT"); r != "" {
		return r
	}
	if b, err := os.ReadFile("/etc/rackbox.conf"); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(l), "root="); ok && v != "" {
				return v
			}
		}
	}
	if fi, err := os.Stat("/volume1"); err == nil && fi.IsDir() {
		return "/volume1/@rackbox"
	}
	return "/var/lib/rackbox"
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)

func imageDir(ref imageRef) string {
	return filepath.Join(root, "images", strings.NewReplacer("/", "_", ":", "_", "@", "_").Replace(ref.String()))
}
func ctDir(name string) string    { return filepath.Join(root, "containers", name) }
func unitName(name string) string { return "rackbox-" + name + ".service" }

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
func writeJSON(p string, v any) error {
	b, _ := json.MarshalIndent(v, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

// ── pull ──
func pull(s string) (imageMeta, error) {
	ref, err := parseRef(s)
	if err != nil {
		return imageMeta{}, err
	}
	c := newClient(ref)
	fmt.Printf("Resolving %s …\n", ref)
	m, plat, err := c.resolve()
	if err != nil {
		return imageMeta{}, err
	}
	dir := imageDir(ref)
	var old imageMeta
	if readJSON(filepath.Join(dir, "meta.json"), &old) == nil && old.Digest == m.Config.Digest {
		fmt.Printf("%s is up to date (%s)\n", ref, plat)
		return old, nil
	}
	tmp := dir + ".pull"
	os.RemoveAll(tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "rootfs"), 0o755); err != nil {
		return imageMeta{}, err
	}
	defer os.RemoveAll(tmp)
	if err := c.blob(m.Config, filepath.Join(tmp, "config.json"), nil); err != nil {
		return imageMeta{}, err
	}
	var total int64
	for _, l := range m.Layers {
		total += l.Size
	}
	fmt.Printf("Image %s for %s: %d layers, %s\n", ref, plat, len(m.Layers), human(total))
	for i, l := range m.Layers {
		f := filepath.Join(tmp, "layer")
		fmt.Printf("[%d/%d] downloading %s …\n", i+1, len(m.Layers), human(l.Size))
		if err := c.blob(l, f, func(n int64) { fmt.Printf("      %s of %s\n", human(n), human(l.Size)) }); err != nil {
			return imageMeta{}, err
		}
		fmt.Printf("[%d/%d] unpacking …\n", i+1, len(m.Layers))
		if err := applyLayer(filepath.Join(tmp, "rootfs"), f); err != nil {
			return imageMeta{}, fmt.Errorf("layer %d: %w", i+1, err)
		}
		os.Remove(f)
	}
	meta := imageMeta{Ref: ref.String(), Digest: m.Config.Digest, Platform: plat, Size: total, Pulled: time.Now()}
	if err := writeJSON(filepath.Join(tmp, "meta.json"), meta); err != nil {
		return imageMeta{}, err
	}
	os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		return imageMeta{}, err
	}
	fmt.Printf("Pulled %s (%s)\n", ref, plat)
	return meta, nil
}

func human(n int64) string {
	u := []string{"B", "KB", "MB", "GB"}
	f, i := float64(n), 0
	for f >= 1024 && i < len(u)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", f, u[i])
}

// ── containers ──
func loadCt(name string) (container, error) {
	var c container
	err := readJSON(filepath.Join(ctDir(name), "container.json"), &c)
	if errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("no container %q", name)
	}
	return c, err
}

func create(c container, setup string) error {
	if !nameRe.MatchString(c.Name) {
		return errors.New("name: lowercase letters, digits and -, max. 31 characters")
	}
	if _, err := os.Stat(ctDir(c.Name)); err == nil {
		return fmt.Errorf("container %q exists already", c.Name)
	}
	ref, err := parseRef(c.Image)
	if err != nil {
		return err
	}
	var meta imageMeta
	if readJSON(filepath.Join(imageDir(ref), "meta.json"), &meta) != nil {
		if meta, err = pull(c.Image); err != nil {
			return err
		}
	}
	for _, b := range c.Binds {
		host, _, ok := strings.Cut(b, ":")
		if !ok || !filepath.IsAbs(host) || strings.Contains(b, "\n") {
			return fmt.Errorf("bind %q: use /host/path:/container/path[:ro]", b)
		}
		os.MkdirAll(host, 0o755)
	}
	c.Image, c.Digest, c.Created = meta.Ref, meta.Digest, time.Now()
	d := ctDir(c.Name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	fmt.Printf("Creating %s from %s …\n", c.Name, c.Image)
	// own copy of the image (no overlayfs needed); reflinks make it instant where the filesystem supports them
	if out, err := exec.Command("cp", "-a", "--reflink=auto", filepath.Join(imageDir(ref), "rootfs"), filepath.Join(d, "rootfs")).CombinedOutput(); err != nil {
		os.RemoveAll(d)
		return fmt.Errorf("copy: %s", out)
	}
	// systemd-nspawn wants something that looks like an OS tree
	os.MkdirAll(filepath.Join(d, "rootfs/usr"), 0o755)
	if _, err := os.Stat(filepath.Join(d, "rootfs/etc/os-release")); err != nil {
		os.MkdirAll(filepath.Join(d, "rootfs/etc"), 0o755)
		os.WriteFile(filepath.Join(d, "rootfs/etc/os-release"), []byte("ID=rackbox\n"), 0o644)
	}
	if err := writeJSON(filepath.Join(d, "container.json"), c); err != nil {
		return err
	}
	if setup != "" {
		fmt.Println("Running setup …")
		cmd := exec.Command("systemd-nspawn", "--quiet", "--register=no", "--resolv-conf=copy-host", "-D", filepath.Join(d, "rootfs"), "/bin/sh", "-c", setup)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}
	return writeUnit(c)
}

// esc quotes one ExecStart argument for systemd
func esc(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%", "$", "$$").Replace(s)
	return `"` + s + `"`
}

func memoryCgroup() bool {
	b, _ := os.ReadFile("/sys/fs/cgroup/cgroup.controllers")
	return strings.Contains(string(b), "memory")
}

func writeUnit(c container) error {
	ref, _ := parseRef(c.Image)
	var ic imageConfig
	_ = readJSON(filepath.Join(imageDir(ref), "config.json"), &ic)
	args := []string{"/usr/bin/systemd-nspawn", "--quiet", "--keep-unit", "--register=no", "--as-pid2", "--link-journal=no",
		"--resolv-conf=copy-host", "--timezone=copy", "--machine=" + c.Name, "-D", filepath.Join(ctDir(c.Name), "rootfs")}
	env := map[string]string{}
	for _, e := range ic.Config.Env {
		if k, v, ok := strings.Cut(e, "="); ok {
			env[k] = v
		}
	}
	for k, v := range c.Env {
		env[k] = v
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--setenv="+k+"="+env[k])
	}
	for _, b := range c.Binds {
		if strings.HasSuffix(b, ":ro") {
			args = append(args, "--bind-ro="+strings.TrimSuffix(b, ":ro"))
		} else {
			args = append(args, "--bind="+strings.TrimSuffix(b, ":rw"))
		}
	}
	if ic.Config.WorkingDir != "" {
		args = append(args, "--chdir="+ic.Config.WorkingDir)
	}
	if u := ic.Config.User; u != "" && u != "root" && u != "0" && u != "0:0" {
		args = append(args, "--user="+strings.Split(u, ":")[0])
	}
	cmd := c.Cmd
	if len(cmd) == 0 {
		cmd = ic.Config.Cmd
	}
	cmd = append(append([]string{}, ic.Config.Entrypoint...), cmd...)
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}
	args = append(append(args, "--"), cmd...)
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = esc(a)
	}
	mem := ""
	if c.Memory != "" && memoryCgroup() {
		mem = "MemoryMax=" + c.Memory + "\n"
	}
	unit := fmt.Sprintf(`# generated by rackbox – change it with rackbox, not by hand
[Unit]
Description=RackBox container %s (%s)
After=network-online.target local-fs.target
Wants=network-online.target
RequiresMountsFor=%s

[Service]
ExecStart=%s
KillMode=mixed
Restart=on-failure
RestartSec=5
%s
[Install]
WantedBy=multi-user.target
`, c.Name, c.Image, root, strings.Join(q, " "), mem)
	if err := os.WriteFile("/etc/systemd/system/"+unitName(c.Name), []byte(unit), 0o644); err != nil {
		return err
	}
	return exec.Command("systemctl", "daemon-reload").Run()
}

func systemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func remove(name string) error {
	if _, err := loadCt(name); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "disable", "--now", unitName(name)).Run()
	os.Remove("/etc/systemd/system/" + unitName(name))
	_ = exec.Command("systemctl", "daemon-reload").Run()
	return os.RemoveAll(ctDir(name)) // bind-mounted data lives outside and is kept
}

type ctStatus struct {
	container
	State  string `json:"state"`
	Memory string `json:"memory_used,omitempty"`
}

func list() []ctStatus {
	out := []ctStatus{}
	ents, _ := os.ReadDir(filepath.Join(root, "containers"))
	for _, e := range ents {
		c, err := loadCt(e.Name())
		if err != nil {
			continue
		}
		st := ctStatus{container: c, State: "stopped"}
		b, _ := exec.Command("systemctl", "show", unitName(c.Name), "-p", "ActiveState,MemoryCurrent").Output()
		for _, l := range strings.Split(string(b), "\n") {
			k, v, _ := strings.Cut(l, "=")
			if k == "ActiveState" {
				st.State = map[string]string{"active": "running", "activating": "starting", "failed": "failed", "deactivating": "stopping"}[v]
				if st.State == "" {
					st.State = "stopped"
				}
			}
			if k == "MemoryCurrent" && v != "" && v != "[not set]" {
				var n int64
				fmt.Sscan(v, &n)
				if n > 0 && n < 1<<50 {
					st.Memory = human(n)
				}
			}
		}
		out = append(out, st)
	}
	return out
}

func images() []imageMeta {
	out := []imageMeta{}
	ents, _ := os.ReadDir(filepath.Join(root, "images"))
	for _, e := range ents {
		var m imageMeta
		if readJSON(filepath.Join(root, "images", e.Name(), "meta.json"), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func printJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

const usage = `RackBox %s – Docker images without a daemon (systemd-nspawn)

  rackbox pull IMAGE                 download an image for this machine (e.g. nginx, ghcr.io/x/y:tag)
  rackbox images [--json]            list images
  rackbox rmi IMAGE                  delete an image
  rackbox create [options] NAME IMAGE [CMD …]
        -e KEY=VALUE   environment (repeatable)
        -v /host:/ct[:ro]  bind a host folder (repeatable)
        -m 256M        memory limit (needs a kernel with memory cgroup)
        --web :8080/   web address shown in RackPanel
        --setup 'sh'   shell commands run once inside the new container
  rackbox start|stop|restart NAME    control a container (start also enables autostart)
  rackbox rm NAME                    delete a container (bound host folders are kept)
  rackbox ls [--json]                list containers
  rackbox logs NAME [-n 200]         container output
  rackbox apps [--json]              app catalog;  rackbox install APP [KEY=VALUE …]
Storage: %s (set root=… in /etc/rackbox.conf)
`

func main() {
	root = storeRoot()
	if len(os.Args) < 2 {
		fmt.Printf(usage, version, root)
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	if cmd != "version" && cmd != "help" && cmd != "-h" && cmd != "--help" && os.Geteuid() != 0 {
		die("must run as root")
	}
	for _, d := range []string{"images", "containers"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	jsonOut := len(args) > 0 && args[len(args)-1] == "--json"
	switch cmd {
	case "version":
		fmt.Println(version)
	case "pull":
		if len(args) != 1 {
			die("usage: rackbox pull IMAGE")
		}
		if _, err := pull(args[0]); err != nil {
			die("%v", err)
		}
	case "images":
		if jsonOut {
			printJSON(images())
			return
		}
		for _, m := range images() {
			fmt.Printf("%-40s %-8s %10s  %s\n", m.Ref, m.Platform, human(m.Size), m.Pulled.Format("2006-01-02 15:04"))
		}
	case "rmi":
		if len(args) != 1 {
			die("usage: rackbox rmi IMAGE")
		}
		ref, err := parseRef(args[0])
		if err != nil {
			die("%v", err)
		}
		for _, c := range list() {
			if c.Image == ref.String() {
				die("image is used by container %s", c.Name)
			}
		}
		if err := os.RemoveAll(imageDir(ref)); err != nil {
			die("%v", err)
		}
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		var envs, binds multi
		fs.Var(&envs, "e", "")
		fs.Var(&binds, "v", "")
		mem := fs.String("m", "", "")
		web := fs.String("web", "", "")
		setup := fs.String("setup", "", "")
		app := fs.String("app", "", "")
		fs.Parse(args)
		if fs.NArg() < 2 {
			die("usage: rackbox create [options] NAME IMAGE [CMD …]")
		}
		c := container{Name: fs.Arg(0), Image: fs.Arg(1), Cmd: fs.Args()[2:], Binds: binds, Memory: *mem, Web: *web, App: *app, Env: map[string]string{}}
		for _, e := range envs {
			k, v, ok := strings.Cut(e, "=")
			if !ok || k == "" {
				die("-e needs KEY=VALUE")
			}
			c.Env[k] = v
		}
		if err := create(c, *setup); err != nil {
			die("%v", err)
		}
		fmt.Printf("Created %s – start it with: rackbox start %s\n", c.Name, c.Name)
	case "start", "stop", "restart":
		if len(args) != 1 {
			die("usage: rackbox %s NAME", cmd)
		}
		if _, err := loadCt(args[0]); err != nil {
			die("%v", err)
		}
		var err error
		switch cmd {
		case "start":
			err = systemctl("enable", "--now", unitName(args[0]))
		case "stop":
			err = systemctl("disable", "--now", unitName(args[0]))
		default:
			err = systemctl("restart", unitName(args[0]))
		}
		if err != nil {
			die("%v", err)
		}
	case "rm":
		if len(args) != 1 {
			die("usage: rackbox rm NAME")
		}
		if err := remove(args[0]); err != nil {
			die("%v", err)
		}
	case "ls", "ps":
		if jsonOut {
			printJSON(list())
			return
		}
		for _, c := range list() {
			fmt.Printf("%-20s %-9s %-36s %s\n", c.Name, c.State, c.Image, c.Memory)
		}
	case "logs":
		if len(args) < 1 {
			die("usage: rackbox logs NAME [-n N]")
		}
		n := "200"
		if len(args) == 3 && args[1] == "-n" {
			n = args[2]
		}
		c := exec.Command("journalctl", "-u", unitName(args[0]), "-n", n, "--no-pager", "-o", "short-iso")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		c.Run()
	case "apps":
		if jsonOut {
			printJSON(loadCatalog())
			return
		}
		for _, a := range loadCatalog() {
			fmt.Printf("%-14s %-30s %s\n", a.ID, a.Image, a.Name)
		}
	case "install":
		if len(args) < 1 {
			die("usage: rackbox install APP [KEY=VALUE …]")
		}
		if err := installApp(args[0], args[1:]); err != nil {
			die("%v", err)
		}
	default:
		fmt.Printf(usage, version, root)
		os.Exit(2)
	}
}
