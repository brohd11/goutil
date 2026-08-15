// Package strutil holds the tiny string helpers that every app in the monorepo ended
// up hand-rolling for itself — plural selection and home-dir expansion — so the next
// app doesn't roll a fourth copy.
package strutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Plural picks the noun form to sit after a count: one when n == 1, many otherwise
// (zero takes many, the usual English convention for "0 items"). It exists because
// three monorepo apps hand-rolled this exact branch; it deliberately returns just the
// word — callers keep their own formatting and embed the result themselves.
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// ExpandHome expands a leading "~" or "~/..." to the current user's home directory
// and passes every other path through unchanged. "~user" forms are rejected rather
// than guessed: only the current user's home shorthand is supported, and accepting
// a literal "~other" as a path is never what the user meant.
//
// The semantics mirror what was gote's normalizeVaultPath, minus that function's
// trim/empty validation and its must-exist-and-be-a-directory check — callers decide
// what the path has to be; this helper only resolves the tilde.
func ExpandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		rest := strings.TrimPrefix(path, "~"+string(filepath.Separator))
		if rest == "~" { // bare "~" — nothing follows the tilde to join
			return home, nil
		}
		return filepath.Join(home, rest), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("only ~/ paths are supported")
	}
	return path, nil
}
