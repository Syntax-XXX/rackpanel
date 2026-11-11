package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type user struct {
	ID          int       `json:"id"`
	Username    string    `json:"username"`
	Hash        string    `json:"password_hash"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	Nickname    string    `json:"nickname"`
	Avatar      string    `json:"avatar"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// public: so wie CasaOS den Benutzer an die Oberfläche gibt (ohne Passwort)
func (u user) public() map[string]any {
	return map[string]any{"id": u.ID, "username": u.Username, "role": u.Role, "email": u.Email,
		"nickname": u.Nickname, "avatar": u.Avatar, "description": u.Description, "created_at": u.CreatedAt}
}

type state struct {
	Secret string   `json:"secret"`
	Users  []user   `json:"users"`
	Keys   []apiKey `json:"keys"`
	NextID int      `json:"next_id"`
}

// apiKey: dauerhafter Schlüssel für fremde Dashboards; gespeichert wird nur der SHA-256
type apiKey struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Hash     string    `json:"hash"`
	ReadOnly bool      `json:"read_only"`
	Created  time.Time `json:"created"`
}

func keyHash(k string) string { h := sha256.Sum256([]byte(k)); return hex.EncodeToString(h[:]) }

type storeT struct {
	mu   sync.Mutex
	path string
	st   state
}

func loadStore(path string) *storeT {
	s := &storeT{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.st)
	}
	if s.st.Secret == "" {
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		s.st.Secret = hex.EncodeToString(buf)
		s.save()
	}
	if s.st.NextID == 0 {
		s.st.NextID = 1
	}
	return s
}

// save: atomar schreiben (Datei nur für root lesbar, enthält Passwort-Hashes und das Token-Geheimnis)
func (s *storeT) save() {
	b, _ := json.MarshalIndent(s.st, "", "  ")
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

func (s *storeT) userByName(name string) *user {
	for i := range s.st.Users {
		if strings.EqualFold(s.st.Users[i].Username, name) {
			return &s.st.Users[i]
		}
	}
	return nil
}

func (s *storeT) userByID(id int) *user {
	for i := range s.st.Users {
		if s.st.Users[i].ID == id {
			return &s.st.Users[i]
		}
	}
	return nil
}

// ── Tokens: HMAC-SHA256, Format base64(json).base64(sig) ─────────────────────
type claims struct {
	UID  int    `json:"uid"`
	Kind string `json:"kind"` // access | refresh
	Exp  int64  `json:"exp"`
}

func (s *storeT) sign(c claims) string {
	b, _ := json.Marshal(c)
	p := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, []byte(s.st.Secret))
	m.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *storeT) verify(tok, kind string) (*user, error) {
	tok = strings.TrimPrefix(strings.TrimSpace(tok), "Bearer ")
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return nil, errors.New("ungültig")
	}
	m := hmac.New(sha256.New, []byte(s.st.Secret))
	m.Write([]byte(parts[0]))
	want := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[1])) {
		return nil, errors.New("Signatur falsch")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var c claims
	if json.Unmarshal(b, &c) != nil || c.Kind != kind || time.Now().Unix() > c.Exp {
		return nil, errors.New("abgelaufen")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.userByID(c.UID)
	if u == nil {
		return nil, errors.New("Benutzer weg")
	}
	cp := *u
	return &cp, nil
}

func (s *storeT) tokens(u *user) map[string]any {
	exp := time.Now().Add(3 * time.Hour)
	return map[string]any{
		"access_token":  s.sign(claims{u.ID, "access", exp.Unix()}),
		"refresh_token": s.sign(claims{u.ID, "refresh", time.Now().Add(14 * 24 * time.Hour).Unix()}),
		"expires_at":    exp.Unix(),
	}
}

// auth: Handler nur mit gültigem Access-Token (Header "Authorization" ohne/mit Bearer, oder ?token=)
// oder mit API-Schlüssel (rp_…, auch über den Header X-API-Key). Nur-Lese-Schlüssel dürfen nur GET.
func (s *storeT) auth(h func(http.ResponseWriter, *http.Request, *user)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("Authorization")
		if tok == "" {
			tok = r.Header.Get("X-API-Key")
		}
		if tok == "" {
			tok = r.URL.Query().Get("token")
		}
		tok = strings.TrimPrefix(strings.TrimSpace(tok), "Bearer ")
		if strings.HasPrefix(tok, "rp_") {
			want := keyHash(tok)
			s.mu.Lock()
			var k *apiKey
			for i := range s.st.Keys {
				if hmac.Equal([]byte(s.st.Keys[i].Hash), []byte(want)) {
					cp := s.st.Keys[i]
					k = &cp
				}
			}
			s.mu.Unlock()
			if k == nil {
				fail(w, http.StatusUnauthorized, 401, "API-Schlüssel ungültig")
				return
			}
			if k.ReadOnly && r.Method != http.MethodGet && r.Method != http.MethodHead {
				fail(w, http.StatusForbidden, 403, "Dieser API-Schlüssel darf nur lesen")
				return
			}
			h(w, r, &user{Username: "key:" + k.Name, Role: "key"})
			return
		}
		u, err := s.verify(tok, "access")
		if err != nil {
			fail(w, http.StatusUnauthorized, 401, "nicht angemeldet")
			return
		}
		h(w, r, u)
	}
}

// session: wie auth, aber nur für angemeldete Menschen – ein API-Schlüssel darf weder Schlüssel noch Passwort ändern
func (s *storeT) session(h func(http.ResponseWriter, *http.Request, *user)) http.HandlerFunc {
	return s.auth(func(w http.ResponseWriter, r *http.Request, u *user) {
		if u.Role == "key" {
			fail(w, http.StatusForbidden, 403, "Nur mit Anmeldung möglich")
			return
		}
		h(w, r, u)
	})
}

// ── Anmelde-Bremse gegen Durchprobieren ──────────────────────────────────────
var loginFails = struct {
	sync.Mutex
	m map[string][]time.Time
}{m: map[string][]time.Time{}}

func tooManyFails(ip string) bool {
	loginFails.Lock()
	defer loginFails.Unlock()
	var keep []time.Time
	for _, t := range loginFails.m[ip] {
		if time.Since(t) < 10*time.Minute {
			keep = append(keep, t)
		}
	}
	loginFails.m[ip] = keep
	return len(keep) >= 8
}

func noteFail(ip string) {
	loginFails.Lock()
	loginFails.m[ip] = append(loginFails.m[ip], time.Now())
	loginFails.Unlock()
}

func clientIP(r *http.Request) string {
	h := r.RemoteAddr
	if i := strings.LastIndex(h, ":"); i > 0 {
		h = h[:i]
	}
	return h
}

// setUser legt den Panel-Benutzer an oder setzt sein Passwort (Installer: rackpanel -set-user NAME, Passwort über stdin)
func (s *storeT) setUser(name, password string) error {
	if !userRe.MatchString(name) || len(password) < 5 {
		return errors.New("Benutzername ungültig oder Passwort kürzer als 5 Zeichen")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if u := s.userByName(name); u != nil {
		u.Hash = string(h)
	} else {
		s.st.Users = append(s.st.Users, user{ID: s.st.NextID, Username: name, Hash: string(h), Role: "admin", CreatedAt: time.Now()})
		s.st.NextID++
	}
	s.save()
	return nil
}

var userRe = regexp.MustCompile(`^[a-z_][a-z0-9_.-]{0,31}$`)

func registerUsers(mux *http.ServeMux, s *storeT) {
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if tooManyFails(ip) {
			fail(w, http.StatusTooManyRequests, 429, "Zu viele Fehlversuche, bitte 10 Minuten warten")
			return
		}
		var in map[string]string
		if readJSON(r, &in) != nil {
			fail(w, 400, 400, "ungültige Anfrage")
			return
		}
		s.mu.Lock()
		u := s.userByName(in["username"])
		var cp user
		if u != nil {
			cp = *u
		}
		s.mu.Unlock()
		if u == nil || bcrypt.CompareHashAndPassword([]byte(cp.Hash), []byte(in["password"])) != nil {
			noteFail(ip)
			fail(w, 401, 401, "Benutzer oder Passwort falsch")
			return
		}
		ok(w, map[string]any{"token": s.tokens(&cp), "user": cp.public()})
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		_ = readJSON(r, &in)
		u, err := s.verify(in["refresh_token"], "refresh")
		if err != nil {
			fail(w, http.StatusUnauthorized, 401, "Sitzung abgelaufen")
			return
		}
		ok(w, s.tokens(u))
	})
	mux.HandleFunc("GET /api/me", s.auth(func(w http.ResponseWriter, r *http.Request, u *user) { ok(w, u.public()) }))
	mux.HandleFunc("PUT /api/me/password", s.session(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in map[string]string
		_ = readJSON(r, &in)
		s.mu.Lock()
		p := s.userByID(u.ID)
		bad := bcrypt.CompareHashAndPassword([]byte(p.Hash), []byte(in["old_password"])) != nil
		s.mu.Unlock()
		if bad {
			fail(w, 400, 400, "Altes Passwort falsch")
			return
		}
		if err := s.setUser(u.Username, in["password"]); err != nil {
			fail(w, 400, 400, err.Error())
			return
		}
		ok(w, "geändert")
	}))

	// API-Schlüssel
	mux.HandleFunc("GET /api/keys", s.session(func(w http.ResponseWriter, r *http.Request, u *user) {
		out := []map[string]any{}
		s.mu.Lock()
		for _, k := range s.st.Keys {
			out = append(out, map[string]any{"id": k.ID, "name": k.Name, "read_only": k.ReadOnly, "created": k.Created})
		}
		s.mu.Unlock()
		ok(w, out)
	}))
	mux.HandleFunc("POST /api/keys", s.session(func(w http.ResponseWriter, r *http.Request, u *user) {
		var in struct {
			Name     string `json:"name"`
			ReadOnly bool   `json:"read_only"`
		}
		if readJSON(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 60 {
			fail(w, 400, 400, "Name fehlt")
			return
		}
		buf := make([]byte, 24)
		_, _ = rand.Read(buf)
		key := "rp_" + hex.EncodeToString(buf)
		s.mu.Lock()
		s.st.Keys = append(s.st.Keys, apiKey{ID: hex.EncodeToString(buf[:4]), Name: strings.TrimSpace(in.Name), Hash: keyHash(key), ReadOnly: in.ReadOnly, Created: time.Now()})
		s.save()
		s.mu.Unlock()
		ok(w, map[string]any{"key": key}) // nur dieses eine Mal sichtbar
	}))
	mux.HandleFunc("DELETE /api/keys/{id}", s.session(func(w http.ResponseWriter, r *http.Request, u *user) {
		s.mu.Lock()
		keep := s.st.Keys[:0]
		for _, k := range s.st.Keys {
			if k.ID != r.PathValue("id") {
				keep = append(keep, k)
			}
		}
		s.st.Keys = keep
		s.save()
		s.mu.Unlock()
		ok(w, "gelöscht")
	}))
}
