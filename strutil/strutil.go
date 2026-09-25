// Package strutil holds small string and path helpers shared across the monorepo's apps.
package strutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Plural returns one when n == 1 and many otherwise (so zero takes many).
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Count renders n with a regular plural noun: "1 file", "3 files".
func Count(n int, noun string) string {
	return strconv.Itoa(n) + " " + Plural(n, noun, noun+"s")
}

// ExpandHome expands a leading "~", "~/..." or "~\..." to the current user's home directory
// and returns any other path unchanged. "~user" forms are an error.
func ExpandHome(path string) (string, error) {
	if path == "~" || len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == '\\') {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		// Accept either separator so examples copied between PowerShell and Unix shells work.
		rest := strings.TrimLeft(path[1:], `/\`)
		rest = strings.NewReplacer("/", string(filepath.Separator), "\\", string(filepath.Separator)).Replace(rest)
		return filepath.Join(home, rest), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("only ~/ or ~\\ paths are supported")
	}
	return path, nil
}

// ContractHome is ExpandHome's inverse: a path at or under home comes back as "~" or
// "~<sep>rest". Anything else, or an empty home, is returned unchanged.
func ContractHome(path, home string) string {
	if home == "" {
		return path
	}
	rel, ok := RelUnder(home, path)
	if !ok {
		return path
	}
	if rel == "." {
		return "~"
	}
	return "~" + string(filepath.Separator) + rel
}

// RelUnder returns path relative to base when path is base itself (rel ".") or inside it.
// ok is false for anything outside base, including when filepath.Rel fails. A child named
// "..foo" is inside.
func RelUnder(base, path string) (rel string, ok bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

// Depth reports how many path segments separate path from base: 0 for the same directory,
// 1 for a direct child. A path Rel cannot relate also reports 0.
func Depth(base, path string) int {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator)) + 1
}
