// Package shellquote renders strings and argument lists for a POSIX shell.
//
// It exists because the same single-quote escape was written three times across this
// workspace: bubblestack/sysopen (building the darwin `cd <dir>` fragment and go-ssh's
// remote command lines) and tmux_s/internal/tmux (rendering an argv into an error
// message). tmux_s could not reuse bubblestack's copy even though it wanted to -- these
// are pure strings functions, but they lived in a package that imports bubbletea, and
// tmux_s is a CLI with no TUI dependency at all. CLI-neutral behavior belongs here.
//
// Two spellings of the same guarantee: Quote/Join always quote, QuoteMinimal/JoinMinimal
// leave a word bare when the shell would read it identically either way. Both are safe;
// the Minimal pair is for text a human reads.
//
// Split is the inverse: it reads a quoted command line back into argv without running a
// shell. It came from configcmd, where it parsed $EDITOR, and moved here when a TUI
// wanted the same parse and could not import a cobra package to get it.
package shellquote

import (
	"fmt"
	"strings"
	"unicode"
)

// Quote wraps s in single quotes as a POSIX single-quoted literal, so nothing inside is
// expanded or word-split. Embedded single quotes are closed, escaped, and reopened --
// the one sequence a single-quoted literal cannot contain directly.
//
// The empty string quotes to ”, which is what keeps an empty argument an argument
// rather than vanishing from the command line.
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

// QuoteMinimal is Quote, except that a word made only of characters the shell treats
// literally is returned unchanged. The result parses to the same word either way; this
// spelling is for command lines a person reads, where quoting every argument is noise.
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

// needsQuote reports whether r forces quoting. The safe set is deliberately narrow:
// alphanumerics plus the punctuation that appears unquoted in ordinary paths, flags and
// key=value arguments. Anything else -- whitespace, globs, quotes, redirections, $ -- is
// quoted rather than reasoned about.
func needsQuote(r rune) bool {
	switch {
	case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return false
	case r == '-', r == '_', r == '.', r == '/', r == ':', r == '=', r == '%', r == '+', r == ',', r == '@':
		return false
	}
	return true
}

// Split parses the shell-style quoting convention commonly used in EDITOR
// (for example `code --wait` or `"/path with spaces/editor" --flag`) without running
// a shell. Backslashes only consume whitespace, a quote, or another backslash, which
// keeps quoted Windows paths intact.
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
