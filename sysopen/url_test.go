package sysopen

import (
	"strings"
	"testing"
)

// Every platform's argv, checked on one host. The Windows arm is the one worth pinning:
// it is NOT pathCommand's (which uses explorer), and its empty third argument is start's
// title parameter — drop it and a quoted URL becomes the window title instead of the
// page that opens.
func TestURLCommand(t *testing.T) {
	const target = "https://example.com/x?a=1"
	tests := []struct {
		goos string
		want []string
	}{
		{"darwin", []string{"open", target}},
		{"windows", []string{"cmd", "/c", "start", "", target}},
		{"linux", []string{"xdg-open", target}},
		{"freebsd", []string{"xdg-open", target}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			cmd := URLCommand(tt.goos, target)
			if len(cmd.Args) != len(tt.want) {
				t.Fatalf("args = %q, want %q", cmd.Args, tt.want)
			}
			for i := range tt.want {
				if cmd.Args[i] != tt.want[i] {
					t.Errorf("arg %d = %q, want %q", i, cmd.Args[i], tt.want[i])
				}
			}
		})
	}
}

// URL and path opening share the darwin and linux arms but must not be conflated:
// Windows reveals a path with explorer and opens a URL through start.
func TestURLAndPathDivergeOnWindows(t *testing.T) {
	urlArgs := URLCommand("windows", "https://example.com").Args
	pathArgs := pathCommand("windows", `C:\tmp\f.txt`, false).Args
	if urlArgs[0] == pathArgs[0] {
		t.Errorf("windows url and path commands both start with %q; they are meant to differ", urlArgs[0])
	}
	if pathArgs[0] != "explorer" {
		t.Errorf("windows path command = %q, want explorer", pathArgs[0])
	}
}

func TestOpenURLRejectsEmpty(t *testing.T) {
	err := OpenURL("")
	if err == nil {
		t.Fatal("empty target should be refused")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error %q should say what was wrong", err)
	}
}
