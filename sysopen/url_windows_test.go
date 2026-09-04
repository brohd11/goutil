//go:build windows

package sysopen

import "testing"

func TestOpenURLPassesQueryStringUnchanged(t *testing.T) {
	old := executeURL
	t.Cleanup(func() { executeURL = old })
	want := "https://example.com/x?a=1&b=two%20words"
	got := ""
	executeURL = func(target string) error {
		got = target
		return nil
	}
	if err := OpenURL(want); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ShellExecute target = %q, want %q", got, want)
	}
}
