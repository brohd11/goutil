package sysopen

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPathCommand(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		path   string
		reveal bool
		want   []string
	}{
		{"darwin open", "darwin", "/tmp/work", false, []string{"open", "/tmp/work"}},
		{"darwin reveal", "darwin", "/tmp/work/a.yml", true, []string{"open", "-R", "/tmp/work/a.yml"}},
		{"windows open", "windows", `C:\work`, false, []string{"explorer", `C:\work`}},
		{"windows reveal", "windows", `C:\work\a.yml`, true, []string{"explorer", `/select,C:\work\a.yml`}},
		{"linux open", "linux", "/tmp/work", false, []string{"xdg-open", "/tmp/work"}},
		{"linux reveal", "linux", "/tmp/work/a.yml", true, []string{"xdg-open", filepath.Dir("/tmp/work/a.yml")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathCommand(tc.goos, tc.path, tc.reveal).Args; !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("argv = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestOpenPathMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	if err := OpenPath(path, false); err == nil {
		t.Fatal("opening a missing path should fail before launching the file manager")
	}
}

func TestOpenPathReportsStartFailure(t *testing.T) {
	oldBuild := buildPathCommand
	t.Cleanup(func() { buildPathCommand = oldBuild })
	buildPathCommand = func(string, bool) *exec.Cmd {
		return exec.Command(filepath.Join(t.TempDir(), "missing-launcher"))
	}
	path := t.TempDir()
	if err := OpenPath(path, false); err == nil {
		t.Fatal("a file-manager start failure should be returned")
	}
	// OpenPath must not mutate or remove its target while probing it.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("target changed: %v", err)
	}
}
