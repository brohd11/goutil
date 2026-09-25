package strutil

import (
	"path/filepath"
	"testing"
)

func TestPlural(t *testing.T) {
	cases := []struct {
		name      string
		n         int
		one, many string
		want      string
	}{
		{"zero takes many", 0, "repo", "repos", "repos"},
		{"one takes one", 1, "repo", "repos", "repo"},
		{"two takes many", 2, "repo", "repos", "repos"},
		{"irregular plural", 5, "entry", "entries", "entries"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Plural(tc.n, tc.one, tc.many); got != tc.want {
				t.Errorf("Plural(%d, %q, %q) = %q, want %q", tc.n, tc.one, tc.many, got, tc.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cases := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"bare tilde", "~", home, false},
		{"slash tilde path", "~/x/nested", filepath.Join(home, "x", "nested"), false},
		{"backslash tilde path", `~\x\nested`, filepath.Join(home, "x", "nested"), false},
		{"other user rejected", "~user/x", "", true},
		{"absolute passthrough", "/var/tmp/x", "/var/tmp/x", false},
		{"relative passthrough", "x/y", "x/y", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandHome(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ExpandHome(%q) = %q, want error", tc.path, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExpandHome(%q) returned error: %v", tc.path, err)
			}
			if got != tc.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	for n, want := range map[int]string{0: "0 files", 1: "1 file", 3: "3 files"} {
		if got := Count(n, "file"); got != want {
			t.Errorf("Count(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestRelUnder(t *testing.T) {
	base := filepath.FromSlash("/a/b")
	cases := []struct {
		path    string
		wantRel string
		wantOK  bool
	}{
		{"/a/b", ".", true},
		{"/a/b/c", "c", true},
		{"/a/b/c/d", filepath.FromSlash("c/d"), true},
		{"/a/b/..foo", "..foo", true},
		{"/a", "", false},
		{"/a/bc", "", false},
		{"/x/y", "", false},
	}
	for _, tc := range cases {
		rel, ok := RelUnder(base, filepath.FromSlash(tc.path))
		if rel != tc.wantRel || ok != tc.wantOK {
			t.Errorf("RelUnder(%q, %q) = (%q, %v), want (%q, %v)", base, tc.path, rel, ok, tc.wantRel, tc.wantOK)
		}
	}
}

func TestContractHome(t *testing.T) {
	home := filepath.FromSlash("/home/bob")
	cases := map[string]string{
		"/home/bob":         "~",
		"/home/bob/notes":   filepath.FromSlash("~/notes"),
		"/home/bobby/notes": filepath.FromSlash("/home/bobby/notes"),
		"/etc":              filepath.FromSlash("/etc"),
	}
	for in, want := range cases {
		if got := ContractHome(filepath.FromSlash(in), home); got != want {
			t.Errorf("ContractHome(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ContractHome("/x", ""); got != "/x" {
		t.Errorf("empty home: got %q", got)
	}
}
