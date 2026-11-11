// RackPanel: lightweight NAS management panel for Debian (Go backend + static web UI).
// Responses look like {success, message, data}.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var version = "dev" // set at build time via -ldflags "-X main.version=…"

var (
	listen  = flag.String("listen", ":80", "Adresse, z. B. :80 oder 0.0.0.0:8080")
	wwwDir  = flag.String("www", "/usr/share/rackpanel/www", "web UI directory")
	dataDir = flag.String("data", "/var/lib/rackpanel", "state directory (panel users, settings)")
	roots   = flag.String("roots", "", "folders for the file manager, comma separated (empty = detect)")
	setUser = flag.String("set-user", "", "create panel user / set password (password on stdin) and exit")
	showVer = flag.Bool("version", false, "Version ausgeben")
)

type result struct {
	Success int    `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func ok(w http.ResponseWriter, data any) { writeJSON(w, 200, result{200, "ok", data}) }

func fail(w http.ResponseWriter, status, code int, msg string) {
	writeJSON(w, status, result{code, msg, nil})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
}

// cors: other dashboards may call the API with a key from the browser.
// Login stays excluded so no foreign page can brute-force passwords.
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/") && p != "/api/login" && p != "/api/refresh" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	flag.Parse()
	if *showVer {
		fmt.Println(version)
		return
	}
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		log.Fatal(err)
	}
	store := loadStore(filepath.Join(*dataDir, "state.json"))
	if *setUser != "" {
		pw, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if err := store.setUser(*setUser, strings.TrimRight(pw, "\r\n")); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println("panel user set:", *setUser)
		return
	}
	if *roots != "" {
		for _, r := range strings.Split(*roots, ",") {
			if r = filepath.Clean(strings.TrimSpace(r)); r != "/" && r != "." {
				fileRoots = append(fileRoots, r)
			}
		}
	} else {
		fileRoots = detectRoots()
	}
	go applySpindown()

	mux := http.NewServeMux()
	registerUsers(mux, store)
	registerSys(mux, store)
	registerFiles(mux, store)
	registerStorage(mux, store)
	registerAdmin(mux, store)
	registerSystem(mux, store)
	registerApps(mux, store)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, 404, "unknown") })

	fs := http.FileServer(http.Dir(*wwwDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Frame-Options", "DENY")
		fs.ServeHTTP(w, r)
	})

	srv := &http.Server{Addr: *listen, Handler: cors(mux), ReadHeaderTimeout: 15 * time.Second}
	log.Printf("RackPanel %s auf %s (Dateien: %s)", version, *listen, strings.Join(fileRoots, ", "))
	log.Fatal(srv.ListenAndServe())
}
