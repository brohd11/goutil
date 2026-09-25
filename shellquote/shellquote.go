// Package shellquote quotes strings and argument lists for a POSIX shell, and splits a
// quoted command line back into argv without running one. Quote/Join always quote;
// QuoteMinimal/JoinMinimal leave shell-safe words bare, for text a person reads.
package shellquote

import (
	"fmt"
	"strings"
	"unicode"
)

// Quote wraps s in single quotes so nothing is expanded or split; embedded single quotes
// are closed, escaped and reopened. An empty s still quotes to a pair of quotes.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Join quotes each argument and joins them with spaces into one command line, so the
// shell that parses it splits the words back exactly where they started.
func Join(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = Quote(a)
	}
	return strings.Join(quoted, " ")
}

// QuoteMinimal is Quote, but returns a word of shell-literal characters unchanged.
func QuoteMinimal(s string) string {
	if s != "" && strings.IndexFunc(s, needsQuote) < 0 {
		return s
	}
	return Quote(s)
}

// JoinMinimal is Join using QuoteMinimal.
func JoinMinimal(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = QuoteMinimal(a)
	}
	return strings.Join(parts, " ")
}

// needsQuote reports whether r forces quoting. The safe set is deliberately narrow.
func needsQuote(r rune) bool {
	switch {
	case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return false
	case r == '-', r == '_', r == '.', r == '/', r == ':', r == '=', r == '%', r == '+', r == ',', r == '@':
		return false
	}
	return true
}

// Split parses shell-style quoting as used in $EDITOR (`code --wait`, `"/a b/ed" -f`)
// without a shell. Backslashes escape only whitespace, quotes and backslashes, so quoted
// Windows paths stay intact.
func Split(raw string) ([]string, error) {
	runes := []rune(raw)
	var args []string
	var word strings.Builder
	var quote rune
	inWord := false
	flush := func() {
		if inWord {
			args = append(args, word.String())
			word.Reset()
			inWord = false
		}
	}
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote == 0 && unicode.IsSpace(r) {
			flush()
			continue
		}
		if r == '\'' {
			if quote == '"' {
				word.WriteRune(r)
				inWord = true
				continue
			}
			if quote == '\'' {
				quote = 0
			} else {
				quote = '\''
				inWord = true
			}
			continue
		}
		if r == '"' {
			if quote == '\'' {
				word.WriteRune(r)
				inWord = true
				continue
			}
			if quote == '"' {
				quote = 0
			} else {
				quote = '"'
				inWord = true
			}
			continue
		}
		if r == '\\' && quote != '\'' && i+1 < len(runes) {
			next := runes[i+1]
			if unicode.IsSpace(next) || next == '\\' || next == '\'' || next == '"' {
				word.WriteRune(next)
				inWord = true
				i++
				continue
			}
		}
		word.WriteRune(r)
		inWord = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote", quote)
	}
	flush()
	return args, nil
}
