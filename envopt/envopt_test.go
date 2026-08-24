package envopt

import (
	"os"
	"strings"
	"testing"
)

const testEnv = "ENVOPT_TEST_DEPTH"

// TestInt pins the rung the environment sits on: below anything typed, above the flag's
// own default. The two easy to get backwards: a typed flag must beat a set variable, and
// a blank variable must read as unset rather than as 0, which is itself a real value.
//
// This table is the one repoview and gote each carried a copy of, merged.
func TestInt(t *testing.T) {
	tests := []struct {
		name        string
		env         string // "" means leave the variable unset entirely
		flagValue   int
		flagChanged bool
		want        int
		wantSet     bool
		wantErr     bool
	}{
		{name: "unset falls through", flagValue: 1, want: 1},
		{name: "blank reads as unset", env: " ", flagValue: 1, want: 1},
		{name: "set supplies the value", env: "2", flagValue: 1, want: 2, wantSet: true},
		{name: "surrounding space is trimmed", env: " 2 ", flagValue: 1, want: 2, wantSet: true},
		{name: "zero is a value", env: "0", flagValue: 1, want: 0, wantSet: true},
		{name: "a typed flag wins", env: "2", flagValue: 4, flagChanged: true, want: 4, wantSet: true},
		{name: "a typed flag wins even unparseable", env: "abc", flagValue: 4, flagChanged: true, want: 4, wantSet: true},
		{name: "not a number", env: "abc", flagValue: 1, want: 1, wantErr: true},
		{name: "negative", env: "-1", flagValue: 1, want: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setenv also unsets for the subtest when the case wants it gone, and
			// restores whatever the developer's own shell had afterwards.
			if tt.env == "" {
				t.Setenv(testEnv, "")
				os.Unsetenv(testEnv)
			} else {
				t.Setenv(testEnv, tt.env)
			}

			got, set, err := Int(testEnv, tt.flagValue, tt.flagChanged)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), testEnv) {
				t.Errorf("error %q does not name %s, so the user cannot tell which knob is wrong", err, testEnv)
			}
			if got != tt.want {
				t.Errorf("value = %d, want %d", got, tt.want)
			}
			if set != tt.wantSet {
				t.Errorf("set = %v, want %v", set, tt.wantSet)
			}
		})
	}
}
