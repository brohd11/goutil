// Package envopt resolves an option a flag may set and an environment variable may
// default: a typed flag wins, then the variable, then the flag's default. It takes the
// flag's value and whether it was set, so it works with any flag package.
package envopt

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Int resolves an integer option. set reports whether the flag or the variable supplied
// it. A malformed or negative variable is an error (returned with the flag value) rather
// than silently ignored, and a blank variable counts as unset.
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
