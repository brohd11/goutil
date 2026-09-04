//go:build !windows

package sysopen

import "testing"

func TestURLCommand(t *testing.T) {
	const target = "https://example.com/x?a=1&b=2"
	for _, tt := range []struct {
		goos string
		want string
	}{{"darwin", "open"}, {"linux", "xdg-open"}, {"freebsd", "xdg-open"}} {
		t.Run(tt.goos, func(t *testing.T) {
			cmd := urlCommand(tt.goos, target)
			if cmd.Args[0] != tt.want || cmd.Args[1] != target {
				t.Fatalf("args = %q, want [%s %s]", cmd.Args, tt.want, target)
			}
		})
	}
}
