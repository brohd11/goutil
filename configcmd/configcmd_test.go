package configcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommandRunsEnsureAndEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	argsFile := filepath.Join(dir, "args")
	t.Setenv("CONFIGCMD_HELPER", "1")
	t.Setenv("CONFIGCMD_HELPER_ARGS", argsFile)
	t.Setenv("EDITOR", `"`+os.Args[0]+`" -test.run=TestEditorHelperProcess -- --wait`)
	t.Setenv("VISUAL", "unused")
	ensured := false
	cmd := NewCommand(Options{
		Path: func() (string, error) { return path, nil },
		Dir:  func() (string, error) { return dir, nil },
		Ensure: func() error {
			ensured = true
			return os.WriteFile(path, []byte("key: value\n"), 0o644)
		},
	})
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ensured {
		t.Fatal("ensure callback was not run")
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "--wait\n"+path+"\n"; got != want {
		t.Fatalf("editor args = %q, want %q", got, want)
	}
}

func TestConfigCommandFallsBackToVisual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	argsFile := filepath.Join(dir, "args")
	t.Setenv("CONFIGCMD_HELPER", "1")
	t.Setenv("CONFIGCMD_HELPER_ARGS", argsFile)
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", `"`+os.Args[0]+`" -test.run=TestEditorHelperProcess --`)
	cmd := NewCommand(Options{Path: func() (string, error) { return path, nil }})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("VISUAL was not run: %v", err)
	}
	if string(data) != path+"\n" {
		t.Fatalf("VISUAL args = %q, want config path", data)
	}
}

func TestEditorArgsRequiresEditorOutsideWindows(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	if _, err := editorArgs("linux"); err == nil || !strings.Contains(err.Error(), "EDITOR") {
		t.Fatalf("error = %v, want an EDITOR/VISUAL explanation", err)
	}
}

func TestEditorArgsFallsBackToNotepadOnWindows(t *testing.T) {
	t.Setenv("EDITOR", " \t")
	t.Setenv("VISUAL", "")
	got, err := editorArgs("windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "notepad.exe" {
		t.Fatalf("editor args = %q, want [notepad.exe]", got)
	}
}

func TestEditorArgsDoesNotReplaceInvalidWindowsEditor(t *testing.T) {
	t.Setenv("EDITOR", `"unterminated`)
	t.Setenv("VISUAL", "")
	if _, err := editorArgs("windows"); err == nil || !strings.Contains(err.Error(), "parse $EDITOR") {
		t.Fatalf("error = %v, want the explicit EDITOR parse failure", err)
	}
}

func TestConfigCommandDir(t *testing.T) {
	oldOpen := openPath
	t.Cleanup(func() { openPath = oldOpen })
	dir := t.TempDir()
	ensured := false
	var opened string
	openPath = func(path string, reveal bool) error {
		opened = path
		if reveal {
			t.Fatal("config directory should be opened directly, not revealed")
		}
		return nil
	}
	cmd := NewCommand(Options{
		Path: func() (string, error) { return filepath.Join(dir, "config.yml"), nil },
		Dir:  func() (string, error) { return dir, nil },
		Ensure: func() error {
			ensured = true
			return nil
		},
	})
	cmd.SetArgs([]string{"--dir"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !ensured || opened != dir {
		t.Fatalf("ensured=%v opened=%q, want true and %q", ensured, opened, dir)
	}
}

func TestConfigCommandEnsureFailureStopsLaunch(t *testing.T) {
	t.Setenv("EDITOR", "does-not-matter")
	cmd := NewCommand(Options{
		Path:   func() (string, error) { t.Fatal("path resolved after ensure failure"); return "", nil },
		Ensure: func() error { return os.ErrPermission },
	})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "ensure config") {
		t.Fatalf("error = %v, want ensure failure", err)
	}
}

func TestConfigCommandReportsEditorFailure(t *testing.T) {
	t.Setenv("CONFIGCMD_HELPER", "1")
	t.Setenv("CONFIGCMD_HELPER_EXIT", "1")
	t.Setenv("EDITOR", `"`+os.Args[0]+`" -test.run=TestEditorHelperProcess --`)
	cmd := NewCommand(Options{Path: func() (string, error) { return "/tmp/config", nil }})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "editor") {
		t.Fatalf("error = %v, want editor exit failure", err)
	}
}

// TestEditorHelperProcess is re-exec'd by the command tests as a portable fake
// editor. It is inert in the parent test process and works on Windows without a shell.
func TestEditorHelperProcess(t *testing.T) {
	if os.Getenv("CONFIGCMD_HELPER") != "1" {
		return
	}
	if os.Getenv("CONFIGCMD_HELPER_EXIT") == "1" {
		os.Exit(7)
	}
	sep := 0
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i + 1
			break
		}
	}
	var body string
	if sep > 0 {
		body = strings.Join(os.Args[sep:], "\n") + "\n"
	}
	if err := os.WriteFile(os.Getenv("CONFIGCMD_HELPER_ARGS"), []byte(body), 0o644); err != nil {
		os.Exit(8)
	}
	os.Exit(0)
}
