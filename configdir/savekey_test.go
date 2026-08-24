package configdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The whole reason SaveKey edits the node tree instead of round-tripping a struct: a
// user's comments and unrelated keys have to survive a setting change. bubblestack pins
// this from its side too; it is pinned here because this is now the implementation.
func TestSaveKeyPreservesCommentsAndOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	const original = `# top of file comment
theme: dark

# why this is set
editor: vim
other_key: keep me
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SaveKey(path, "theme", "light", nil); err != nil {
		t.Fatalf("SaveKey: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, "theme: light") {
		t.Errorf("theme was not updated:\n%s", got)
	}
	for _, want := range []string{
		"# top of file comment",
		"# why this is set",
		"editor: vim",
		"other_key: keep me",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SaveKey dropped %q:\n%s", want, got)
		}
	}
}

// An absent key is appended rather than replacing the document.
func TestSaveKeyAppendsMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("existing: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveKey(path, "added", "yes", nil); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(path)
	got := string(out)
	if !strings.Contains(got, "existing: value") || !strings.Contains(got, "added: yes") {
		t.Errorf("want both keys, got:\n%s", got)
	}
}

// The two missing-file behaviors that were the only difference between the copies this
// replaced: nil seeds an empty document, a value seeds the app's defaults alongside.
func TestSaveKeyMissingFile(t *testing.T) {
	t.Run("nil seed writes only the key", func(t *testing.T) {
		// A nested path also proves the parent directory is created.
		path := filepath.Join(t.TempDir(), "nested", "config.yml")
		if err := SaveKey(path, "theme", "dark", nil); err != nil {
			t.Fatal(err)
		}
		out, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("file not created: %v", err)
		}
		got := strings.TrimSpace(string(out))
		if got != "theme: dark" {
			t.Errorf("got %q, want only the one key", got)
		}
	})

	t.Run("seed writes the defaults too", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yml")
		seed := struct {
			Theme   string `yaml:"theme"`
			Editor  string `yaml:"editor"`
			Verbose bool   `yaml:"verbose"`
		}{Theme: "dark", Editor: "vim"}

		if err := SaveKey(path, "theme", "light", seed); err != nil {
			t.Fatal(err)
		}
		out, _ := os.ReadFile(path)
		got := string(out)
		if !strings.Contains(got, "theme: light") {
			t.Errorf("seeded file did not take the new value:\n%s", got)
		}
		if !strings.Contains(got, "editor: vim") {
			t.Errorf("seeded file lost the other defaults:\n%s", got)
		}
	})
}
