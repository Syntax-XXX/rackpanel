package main

import "testing"

func TestParseDiskstats(t *testing.T) {
	in := `   8       0 sda 1000 0 2048 0 500 0 4096 0 0 0 0
   8       1 sda1 10 0 20 0 5 0 40 0 0 0 0
   9       0 md0 7 0 1 0 3 0 2 0 0 0 0
   7       0 loop0 1 0 1 0 1 0 1 0 0 0 0
  31       0 mtdblock0 1 0 1 0 1 0 1 0 0 0 0`
	got := parseDiskstats(in)
	if len(got) != 2 || got[0] != (diskIO{"sda", 2048 * 512, 4096 * 512}) || got[1] != (diskIO{"md0", 512, 1024}) {
		t.Fatalf("got %+v", got)
	}
}

func TestSsoAllowed(t *testing.T) {
	cases := []struct {
		ip, list string
		want     bool
	}{
		{"10.40.32.69", "10.40.32.69", true},
		{"10.40.32.69", " 127.0.0.1 , 10.40.32.69", true},
		{"10.40.32.70", "10.40.32.69", false},
		{"10.40.32.69", "", false},
		{"[::1]", "::1", true},
	}
	for _, c := range cases {
		if got := ssoAllowed(c.ip, c.list); got != c.want {
			t.Errorf("ssoAllowed(%q, %q) = %v, want %v", c.ip, c.list, got, c.want)
		}
	}
}
