package executil

import (
	"context"
	"strings"
	"testing"
)

func TestCommandRejectsEmptyArgv(t *testing.T) {
	if _, err := Command(); err == nil {
		t.Fatal("Command() should reject an empty argv")
	}
	if _, err := CommandContext(context.Background(), " "); err == nil {
		t.Fatal("CommandContext should reject a blank program")
	}
}

func TestCmdQuotePreservesBoundaries(t *testing.T) {
	for _, input := range []string{"", "plain", "two words", `C:\\Program Files\\tool`, `a&b|c`, `say \"hi\"`, `trail\\`} {
		got := cmdQuote(input, false)
		if !strings.HasPrefix(got, `^"`) || !strings.HasSuffix(got, `^"`) {
			t.Errorf("CmdQuote(%q) = %q, want quoted argument", input, got)
		}
	}
}

func TestCmdJoinRejectsNewlineInjection(t *testing.T) {
	if _, err := CmdJoin([]string{"tool", "safe\nwhoami"}); err == nil {
		t.Fatal("CmdJoin should reject a newline crossing the cmd.exe boundary")
	}
}
