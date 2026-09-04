// Package strutil holds the tiny string and path helpers that every app in the monorepo
// ended up hand-rolling for itself — plural selection, home-dir expansion, path depth —
// so the next app doesn't roll a fourth copy.
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

// ExpandHome expands a leading "~", "~/...", or "~\\..." to the current user's home directory
// and passes every other path through unchanged. "~user" forms are rejected rather
// than guessed: only the current user's home shorthand is supported, and accepting
// a literal "~other" as a path is never what the user meant.
//
// The semantics mirror what was gote's normalizeVaultPath, minus that function's
// trim/empty validation and its must-exist-and-be-a-directory check — callers decide
// what the path has to be; this helper only resolves the tilde.
func ExpandHome(path string) (string, error) {
	if path == "~" || len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == '\\') {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		rest := strings.TrimLeft(path[1:], `/\`)
		// Tilde paths are user-facing rather than native filesystem literals. Treat
		// both separator spellings consistently so copied examples work unchanged
		// in PowerShell and Unix shells.
		rest = strings.NewReplacer("/", string(filepath.Separator), "\\", string(filepath.Separator)).Replace(rest)
		return filepath.Join(home, rest), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("only ~/ or ~\\ paths are supported")
	}
	return path, nil
}

// Depth reports how many path segments separate path from base: 0 when they are the same
// directory, 1 for a direct child, and so on. An unrelated path (one filepath.Rel cannot
// express) is also 0, so a caller walking a tree treats it as the root rather than
// letting a negative or garbage depth through.
//
// It exists because four copies of this arithmetic were in circulation — gitstack,
// gdaddon twice (once inlined into a walk), and gote under the name dirDepth. A
// depth-limited walk that is off by one silently stops finding things, which is the kind
// of bug that gets noticed months later as "it missed a repo".
func Depth(base, path string) int {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator)) + 1
}
