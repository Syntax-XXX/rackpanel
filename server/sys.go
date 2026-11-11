package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ── Messwerte aus /proc ──────────────────────────────────────────────────────

var cpuPrev struct {
	sync.Mutex
	idle, total uint64
	percent     float64
}

// cpuSample: Auslastung seit dem letzten Aufruf (wird vom Sammler alle 3 s aufgerufen)
func cpuSample() float64 {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	f := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
	var total, idle uint64
	for i, v := range f[1:] {
		n, _ := strconv.ParseUint(v, 10, 64)
		total += n
		if i == 3 || i == 4 { // idle + iowait
			idle += n
		}
	}
	cpuPrev.Lock()
	defer cpuPrev.Unlock()
	if cpuPrev.total > 0 && total > cpuPrev.total {
		dt, di := total-cpuPrev.total, idle-cpuPrev.idle
		cpuPrev.percent = float64(int(1000*float64(dt-di)/float64(dt))) / 10
	}
	cpuPrev.idle, cpuPrev.total = idle, total
	return cpuPrev.percent
}

func memInfo() map[string]any {
	m := map[string]uint64{}
	f, err := os.Open("/proc/meminfo")
	if err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			p := strings.Fields(sc.Text())
			if len(p) >= 2 {
				n, _ := strconv.ParseUint(p[1], 10, 64)
				m[strings.TrimSuffix(p[0], ":")] = n * 1024
			}
		}
		f.Close()
	}
	total, avail := m["MemTotal"], m["MemAvailable"]
	used := total - avail
	pct := 0.0
	if total > 0 {
		pct = float64(int(1000*float64(used)/float64(total))) / 10
	}
	return map[string]any{"total": total, "available": avail, "used": used, "free": m["MemFree"],
		"usedPercent": pct, "swapTotal": m["SwapTotal"], "swapFree": m["SwapFree"]}
}

type netStat struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytesSent"`
	BytesRecv   uint64 `json:"bytesRecv"`
	PacketsSent uint64 `json:"packetsSent"`
	PacketsRecv uint64 `json:"packetsRecv"`
	State       string `json:"state"`
	Time        int64  `json:"time"`
}

func netStats() []netStat {
	out := []netStat{}
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return out
	}
	for _, l := range strings.Split(string(b), "\n")[2:] {
		p := strings.Fields(strings.Replace(l, ":", " ", 1))
		if len(p) < 17 || p[0] == "lo" || strings.HasPrefix(p[0], "veth") || strings.HasPrefix(p[0], "docker") || strings.HasPrefix(p[0], "br-") {
			continue
		}
		u := func(i int) uint64 { n, _ := strconv.ParseUint(p[i], 10, 64); return n }
		st, _ := os.ReadFile("/sys/class/net/" + p[0] + "/operstate")
		out = append(out, netStat{p[0], u(9), u(1), u(10), u(2), strings.TrimSpace(string(st)), time.Now().Unix()})
	}
	return out
}

// volumes: alle eingehängten Block-Dateisysteme (/, /volumeN, /mnt/… ) – Größe/Belegung in Bytes
type volume struct {
	Path  string `json:"path"`
	Dev   string `json:"dev"`
	FS    string `json:"fs"`
	Size  uint64 `json:"size"`
	Used  uint64 `json:"used"`
	Avail uint64 `json:"avail"`
}

func volumes() []volume {
	out := []volume{}
	seen := map[string]bool{}
	b, _ := os.ReadFile("/proc/mounts")
	for _, l := range strings.Split(string(b), "\n") {
		p := strings.Fields(l)
		if len(p) < 3 || !strings.HasPrefix(p[0], "/dev/") || seen[p[0]] || strings.HasPrefix(p[1], "/boot") ||
			strings.HasPrefix(p[0], "/dev/loop") || strings.HasPrefix(p[2], "fuse") || p[2] == "squashfs" || p[2] == "vfat" && p[1] != "/" {
			continue
		}
		var st syscall.Statfs_t
		if syscall.Statfs(p[1], &st) != nil {
			continue
		}
		seen[p[0]] = true
		if p[0] == "/dev/root" { // Kernel ohne initrd meldet root so; echtes Gerät steht in der Kommandozeile
			c, _ := os.ReadFile("/proc/cmdline")
			for _, f := range strings.Fields(string(c)) {
				if strings.HasPrefix(f, "root=/dev/") {
					p[0] = f[5:]
				}
			}
		}
		bs := uint64(st.Bsize)
		out = append(out, volume{p[1], p[0], p[2], st.Blocks * bs, (st.Blocks - st.Bfree) * bs, st.Bavail * bs})
	}
	return out
}

// detectRoots: Bereiche für den Datei-Manager – alle Daten-Volumes außer /, dazu /home, /srv, /mnt, /media
func detectRoots() []string {
	var r []string
	for _, v := range volumes() {
		if v.Path != "/" {
			r = append(r, v.Path)
		}
	}
	for _, d := range []string{"/home", "/srv", "/mnt", "/media"} {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			dup := false
			for _, x := range r {
				dup = dup || x == d
			}
			if !dup {
				r = append(r, d)
			}
		}
	}
	return r
}

// ── Platten: SMART (ohne schlafende Platten zu wecken), alle 60 s aufgefrischt ──
type diskInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Model    string `json:"model"`
	Serial   string `json:"serial"`
	Size     uint64 `json:"size"`
	Temp     int    `json:"temperature"`
	Health   bool   `json:"health"`
	Standby  bool   `json:"standby"`
	Hours    int    `json:"power_on_hours"`
	Realloc  int    `json:"reallocated"`
	Pending  int    `json:"pending"`
	Rotation string `json:"rotation"`
}

var disksCache struct {
	sync.Mutex
	list []diskInfo
	at   time.Time
}

func readDisks() []diskInfo {
	disksCache.Lock()
	defer disksCache.Unlock()
	if time.Since(disksCache.at) < 60*time.Second && disksCache.list != nil {
		return disksCache.list
	}
	out := []diskInfo{}
	ents, _ := os.ReadDir("/sys/block")
	for _, e := range ents {
		n := e.Name()
		if !strings.HasPrefix(n, "sd") && !strings.HasPrefix(n, "nvme") {
			continue
		}
		d := diskInfo{Name: n, Path: "/dev/" + n, Health: true}
		if b, err := os.ReadFile("/sys/block/" + n + "/size"); err == nil {
			s, _ := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
			d.Size = s * 512
		}
		if d.Size == 0 { // leerer Kartenleser
			continue
		}
		if b, err := os.ReadFile("/sys/block/" + n + "/device/model"); err == nil {
			d.Model = strings.TrimSpace(string(b))
		}
		// -n standby: schlafende Platte nicht aufwecken (Exit-Code 2)
		cmd := exec.Command("smartctl", "-n", "standby", "-j", "-H", "-i", "-A", d.Path)
		raw, err := cmd.Output()
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode()&2 != 0 && len(raw) < 200 {
			d.Standby = true
		}
		var sm struct {
			SerialNumber string `json:"serial_number"`
			ModelName    string `json:"model_name"`
			RotationRate int    `json:"rotation_rate"`
			SmartStatus  *struct {
				Passed bool `json:"passed"`
			} `json:"smart_status"`
			Temperature struct {
				Current int `json:"current"`
			} `json:"temperature"`
			PowerOnTime struct {
				Hours int `json:"hours"`
			} `json:"power_on_time"`
			Ata struct {
				Table []struct {
					ID  int `json:"id"`
					Raw struct {
						Value int `json:"value"`
					} `json:"raw"`
				} `json:"table"`
			} `json:"ata_smart_attributes"`
		}
		if json.Unmarshal(raw, &sm) == nil {
			if sm.ModelName != "" {
				d.Model = sm.ModelName
			}
			d.Serial = sm.SerialNumber
			d.Temp = sm.Temperature.Current
			d.Hours = sm.PowerOnTime.Hours
			if sm.SmartStatus != nil {
				d.Health = sm.SmartStatus.Passed
			}
			if sm.RotationRate > 0 {
				d.Rotation = strconv.Itoa(sm.RotationRate) + " U/min"
			}
			for _, a := range sm.Ata.Table {
				switch a.ID {
				case 5:
					d.Realloc = a.Raw.Value
				case 197:
					d.Pending = a.Raw.Value
				}
			}
			if d.Realloc > 0 || d.Pending > 0 {
				d.Health = false
			}
		}
		out = append(out, d)
	}
	disksCache.list, disksCache.at = out, time.Now()
	return out
}

func maxDiskTemp() int {
	t := 0
	for _, d := range readDisks() {
		if d.Temp > t {
			t = d.Temp
		}
	}
	return t
}

func cpuTemp() int {
	ents, _ := os.ReadDir("/sys/class/hwmon")
	for _, e := range ents {
		base := "/sys/class/hwmon/" + e.Name() + "/"
		n, _ := os.ReadFile(base + "name")
		switch strings.TrimSpace(string(n)) {
		case "coretemp", "k10temp", "cpu_thermal", "soc_thermal", "zenpower":
			if b, err := os.ReadFile(base + "temp1_input"); err == nil {
				v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
				return v / 1000
			}
		}
	}
	if b, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		return v / 1000
	}
	return 0
}

func fanInfo() map[string]any {
	out := map[string]any{"rpm": 0, "pwm": 0}
	ents, _ := os.ReadDir("/sys/class/hwmon")
	for _, e := range ents {
		base := "/sys/class/hwmon/" + e.Name() + "/"
		if n, _ := os.ReadFile(base + "name"); strings.TrimSpace(string(n)) == "gpio_fan" {
			rd := func(f string) int {
				b, _ := os.ReadFile(base + f)
				v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
				return v
			}
			out["rpm"], out["pwm"], out["max"] = rd("fan1_input"), rd("pwm1"), rd("fan1_max")
		}
	}
	return out
}

func hostname() string { h, _ := os.Hostname(); return h }

func uptimeSeconds() int64 {
	b, _ := os.ReadFile("/proc/uptime")
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return int64(v)
}

func loadAvg() string {
	b, _ := os.ReadFile("/proc/loadavg")
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return ""
	}
	return strings.Join(f[:3], " ")
}

func registerSys(mux *http.ServeMux, s *storeT) {
	go func() { // CPU-Messung im Hintergrund, damit die Prozentwerte stimmen
		for {
			cpuSample()
			time.Sleep(3 * time.Second)
		}
	}()
	// Live-Werte (alle 3 s abgefragt)
	mux.HandleFunc("GET /api/stats", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		ok(w, map[string]any{"cpu": cpuPrev.percent, "mem": memInfo(), "net": netStats(), "load": loadAvg(), "uptime": uptimeSeconds(), "time": time.Now().Unix()})
	}))
	mux.HandleFunc("GET /api/overview", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		kern, _ := os.ReadFile("/proc/sys/kernel/osrelease")
		ok(w, map[string]any{"hostname": hostname(), "uptime": uptimeSeconds(), "load": loadAvg(), "os": osName(),
			"kernel": strings.TrimSpace(string(kern)), "model": model(), "arch": runtime.GOARCH, "cpus": runtime.NumCPU(),
			"cpu_temp": cpuTemp(), "disk_temp": maxDiskTemp(), "fan": fanInfo(), "disks": readDisks(), "volumes": volumes(),
			"raid": mdstat(), "net": netStats(), "mem": memInfo(), "version": version})
	}))
}

func osName() string {
	b, _ := os.ReadFile("/etc/os-release")
	for _, l := range strings.Split(string(b), "\n") {
		if v, f := strings.CutPrefix(l, "PRETTY_NAME="); f {
			return strings.Trim(v, `"`)
		}
	}
	return "Linux"
}

// model: Gerätename aus Device Tree (ARM) oder DMI (PC)
func model() string {
	for _, f := range []string{"/proc/device-tree/model", "/sys/class/dmi/id/product_name"} {
		if b, err := os.ReadFile(f); err == nil && len(b) > 0 {
			return strings.TrimSpace(strings.Trim(string(b), "\x00"))
		}
	}
	return ""
}
