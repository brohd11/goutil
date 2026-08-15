package configdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := Dir("myapp")
	if err != nil {
		t.Fatalf("Dir returned error: %v", err)
	}
	want := filepath.Join(home, ".myapp")
	if got != want {
		t.Errorf("Dir(\"myapp\") = %q, want %q", got, want)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yml")
	var v struct {
		Name string `yaml:"name"`
	}
	if err := Load(path, &v); err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if v.Name != "" {
		t.Errorf("Load on missing file populated v = %+v, want zero value", v)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	type cfg struct {
		Name  string `yaml:"name"`
		Count int    `yaml:"count"`
	}
	in := cfg{Name: "gote", Count: 3}

	if err := SaveAtomic(dir, "config.yml", in); err != nil {
		t.Fatalf("SaveAtomic returned error: %v", err)
	}
	var out cfg
	if err := Load(filepath.Join(dir, "config.yml"), &out); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v, want %+v", out, in)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("\tnot: [valid"), 0o644); err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := Load(path, &v); err == nil {
		t.Fatal("Load on malformed YAML returned nil error, want parse error")
	}
}

func TestSaveAtomicCreatesDirAndOverwrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cfgdir")
	type cfg struct {
		Name string `yaml:"name"`
	}

	if err := SaveAtomic(dir, "config.yml", cfg{Name: "first"}); err != nil {
		t.Fatalf("SaveAtomic returned error: %v", err)
	}
	// A second save over the existing file must still succeed — the temp+rename
	// dance leaves the previous file in place only on failure, never blocking it.
	if err := SaveAtomic(dir, "config.yml", cfg{Name: "second"}); err != nil {
		t.Fatalf("SaveAtomic over existing file returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.yml"))
	if err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
	if got := string(data); got != "name: second\n" {
		t.Errorf("saved content = %q, want %q", got, "name: second\n")
	}
	info, err := os.Stat(filepath.Join(dir, "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("saved file mode = %o, want 644", perm)
	}
	// The temp file must be cleaned up, not left littering the config dir.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.yml" {
		t.Errorf("dir contains %v, want exactly [config.yml]", entries)
	}
}
