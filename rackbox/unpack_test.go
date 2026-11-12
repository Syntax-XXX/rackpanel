package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureJoinStaysInside(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "usr/lib"), 0o755)
	os.Symlink("/", filepath.Join(root, "escape"))       // absolute -> container root
	os.Symlink("../../../..", filepath.Join(root, "up")) // relative climb
	os.Symlink("usr/lib", filepath.Join(root, "lib"))    // normal usrmerge link
	cases := map[string]string{
		"escape/etc/passwd": "etc/passwd",
		"up/etc/shadow":     "etc/shadow",
		"../../etc/hosts":   "etc/hosts",
		"lib/x.so":          "usr/lib/x.so",
		"lib":               "lib", // last element is not followed
	}
	for in, want := range cases {
		got, err := secureJoin(root, in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != filepath.Join(root, want) {
			t.Errorf("%s -> %s, want %s", in, got, filepath.Join(root, want))
		}
	}
}

func TestParseRef(t *testing.T) {
	for in, want := range map[string]string{
		"nginx":                   "registry-1.docker.io library/nginx latest",
		"arm32v5/debian:bookworm": "registry-1.docker.io arm32v5/debian bookworm",
		"ghcr.io/a/b:1.2":         "ghcr.io a/b 1.2",
		"localhost:5000/x":        "localhost:5000 x latest",
	} {
		r, err := parseRef(in)
		if err != nil || r.Registry+" "+r.Repo+" "+r.Tag != want {
			t.Errorf("%s -> %+v %v", in, r, err)
		}
	}
	if _, err := parseRef("bad name;rm"); err == nil {
		t.Error("accepted invalid ref")
	}
}

func TestStopSignal(t *testing.T) {
	for in, want := range map[string]string{"": "SIGTERM", "SIGQUIT": "SIGQUIT", "quit": "SIGQUIT", "SIGINT": "SIGINT", "15": "SIGTERM", "SIGRTMIN+3": "SIGTERM", "X; rm": "SIGTERM"} {
		if got := stopSignal(in); got != want {
			t.Errorf("stopSignal(%q) = %q, want %q", in, got, want)
		}
	}
}
