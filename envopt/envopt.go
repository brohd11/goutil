// Package envopt resolves a command-line option that a flag may set and an environment
// variable may default.
//
// The ladder is always the same: an explicitly passed flag wins; otherwise the variable
// is consulted; otherwise the flag's own default stands. It lives here because repoview
// and gote had written it out identically -- same body, same doc comment, and the same
// nine-case test table each -- for REPOVIEW_DEPTH and GOTE_DEPTH.
//
// Deliberately free of any CLI framework: it takes the flag's value and whether the user
// actually set it, which every flag package can answer (cobra: Flags().Changed(name)).
// Putting it behind cobra would repeat the mistake that stranded shell quoting inside a
// bubbletea package.
package envopt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Int resolves an integer option. set reports whether the value came from the flag or
// the environment rather than falling through to the flag's default, which callers use
// to tell "the user asked for this" from "nobody said".
//
// A malformed or negative value is refused rather than ignored: the variable normally
// lives in a shell profile, where a silent fallback to the default would be diagnosed as
// "the setting does nothing" long after the typo was made. On error the flag value is
// returned alongside, so a caller that chooses to warn and continue still has something
// usable.
//
// An unset variable and one set to whitespace are both treated as absent -- FOO= in a
// profile is how a shell user disables a setting.
func Int(envName string, flagValue int, flagChanged bool) (value int, set bool, err error) {
	if flagChanged {
		return flagValue, true, nil
	}
	raw, ok := os.LookupEnv(envName)
	if !ok {
		return flagValue, false, nil
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return flagValue, false, nil
	}
	n, convErr := strconv.Atoi(trimmed)
	if convErr != nil {
		return flagValue, false, fmt.Errorf("%s %q is not a number", envName, raw)
	}
	if n < 0 {
		return flagValue, false, fmt.Errorf("%s %d is negative", envName, n)
	}
	return n, true, nil
}
