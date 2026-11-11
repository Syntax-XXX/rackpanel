package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// imageRef: registry/repository:tag (Docker Hub defaults like the docker CLI)
type imageRef struct{ Registry, Repo, Tag string }

func (r imageRef) String() string {
	s := r.Repo + ":" + r.Tag
	if r.Registry != "registry-1.docker.io" {
		s = r.Registry + "/" + s
	}
	return strings.TrimPrefix(s, "library/")
}

var refRe = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(:[0-9]+)?(/[a-z0-9]+([._-][a-z0-9]+)*)*(:[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}|@sha256:[a-f0-9]{64})?$`)

func parseRef(s string) (imageRef, error) {
	if !refRe.MatchString(s) {
		return imageRef{}, fmt.Errorf("invalid image name %q", s)
	}
	r := imageRef{Registry: "registry-1.docker.io", Tag: "latest"}
	if i := strings.LastIndex(s, "@"); i > 0 {
		r.Tag, s = s[i+1:], s[:i]
	} else if i := strings.LastIndex(s, ":"); i > strings.LastIndex(s, "/") {
		r.Tag, s = s[i+1:], s[:i]
	}
	if first, rest, ok := strings.Cut(s, "/"); ok && (strings.ContainsAny(first, ".:") || first == "localhost") {
		r.Registry, s = first, rest
		if r.Registry == "docker.io" {
			r.Registry = "registry-1.docker.io"
		}
	}
	if r.Registry == "registry-1.docker.io" && !strings.Contains(s, "/") {
		s = "library/" + s
	}
	r.Repo = s
	return r, nil
}

type client struct {
	ref   imageRef
	token string
	http  *http.Client
}

func newClient(ref imageRef) *client {
	return &client{ref: ref, http: &http.Client{Timeout: 0}}
}

// do sends a registry request and fetches an anonymous pull token on the first 401
func (c *client) do(path, accept string) (*http.Response, error) {
	for try := 0; try < 2; try++ {
		req, _ := http.NewRequest("GET", "https://"+c.ref.Registry+"/v2/"+c.ref.Repo+path, nil)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == 401 && try == 0 {
			ch := resp.Header.Get("WWW-Authenticate")
			resp.Body.Close()
			if err := c.auth(ch); err != nil {
				return nil, err
			}
			continue
		}
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
			resp.Body.Close()
			return nil, fmt.Errorf("registry: %s %s", resp.Status, strings.TrimSpace(string(b)))
		}
		return resp, nil
	}
	return nil, errors.New("registry: not authorized")
}

func (c *client) auth(challenge string) error {
	params := map[string]string{}
	for _, m := range regexp.MustCompile(`(\w+)="([^"]*)"`).FindAllStringSubmatch(challenge, -1) {
		params[m[1]] = m[2]
	}
	if !strings.HasPrefix(challenge, "Bearer") || params["realm"] == "" {
		return errors.New("registry: unsupported authentication")
	}
	u := params["realm"] + "?service=" + params["service"] + "&scope=repository:" + c.ref.Repo + ":pull"
	resp, err := c.http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var t struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return err
	}
	c.token = t.Token
	if c.token == "" {
		c.token = t.AccessToken
	}
	return nil
}

type descriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	Platform  *struct {
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
		Variant      string `json:"variant"`
	} `json:"platform"`
}

type manifest struct {
	MediaType string       `json:"mediaType"`
	Manifests []descriptor `json:"manifests"`
	Config    descriptor   `json:"config"`
	Layers    []descriptor `json:"layers"`
}

const acceptAll = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, " +
	"application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"

// hostPlatform: what this machine can run, best match first (an ARMv7 can run v6 and v5 images, not the reverse)
func hostPlatform() (arch string, variants []string) {
	arch = runtime.GOARCH
	if arch != "arm" {
		return arch, []string{"", "v8"}
	}
	v := 7
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if m := regexp.MustCompile(`CPU architecture:\s*(\d+)`).FindSubmatch(b); m != nil {
			fmt.Sscan(string(m[1]), &v)
		}
	}
	for ; v >= 5; v-- {
		variants = append(variants, fmt.Sprintf("v%d", v))
	}
	return arch, variants
}

// resolve: manifest for this machine (picks the right entry from a multi-arch index)
func (c *client) resolve() (*manifest, string, error) {
	resp, err := c.do("/manifests/"+c.ref.Tag, acceptAll)
	if err != nil {
		return nil, "", err
	}
	var m manifest
	err = json.NewDecoder(resp.Body).Decode(&m)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}
	if len(m.Manifests) == 0 {
		return &m, "", nil // single-platform image: trust it
	}
	arch, variants := hostPlatform()
	var pick *descriptor
	var have []string
	for _, v := range variants {
		for i, d := range m.Manifests {
			if d.Platform == nil || d.Platform.OS != "linux" {
				continue
			}
			if d.Platform.Architecture == arch && d.Platform.Variant == v && pick == nil {
				pick = &m.Manifests[i]
			}
		}
	}
	if pick == nil {
		for _, d := range m.Manifests {
			if d.Platform != nil && d.Platform.OS == "linux" {
				have = append(have, strings.TrimSuffix(d.Platform.Architecture+"/"+d.Platform.Variant, "/"))
			}
		}
		return nil, "", fmt.Errorf("no image for %s/%s (available: %s)", arch, strings.Join(variants, ","), strings.Join(have, " "))
	}
	resp, err = c.do("/manifests/"+pick.Digest, acceptAll)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	var im manifest
	if err := json.NewDecoder(resp.Body).Decode(&im); err != nil {
		return nil, "", err
	}
	plat := arch
	if pick.Platform.Variant != "" {
		plat += "/" + pick.Platform.Variant
	}
	return &im, plat, nil
}

// blob downloads a blob to a file and checks its sha256
func (c *client) blob(d descriptor, dst string, progress func(int64)) error {
	resp, err := c.do("/blobs/"+d.Digest, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	h := sha256.New()
	var n int64
	buf := make([]byte, 256<<10)
	last := time.Now()
	for {
		k, rerr := resp.Body.Read(buf)
		if k > 0 {
			if _, err := f.Write(buf[:k]); err != nil {
				f.Close()
				return err
			}
			h.Write(buf[:k])
			n += int64(k)
			if progress != nil && time.Since(last) > 2*time.Second {
				progress(n)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if got := "sha256:" + hex.EncodeToString(h.Sum(nil)); got != d.Digest {
		return fmt.Errorf("checksum mismatch for %s", d.Digest)
	}
	return nil
}
