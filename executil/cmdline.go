package executil

import (
	"fmt"
	"strings"
)

const cmdMeta = "()[]%!^\"" + "`" + `<>&|;,*? `

// cmdQuote renders one argument for cmd.exe. The result survives both cmd's
// metacharacter pass and the Windows argv parser used by the child process.
// doubleMeta is needed when cmd invokes a batch file, where the line is parsed
// once by cmd and again by the batch-file machinery.
func cmdQuote(arg string, doubleMeta bool) string {
	var quoted strings.Builder
	quoted.Grow(len(arg) + 2)
	quoted.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			quoted.WriteString(strings.Repeat(`\`, backslashes*2+1))
			quoted.WriteRune(r)
			backslashes = 0
			continue
		}
		quoted.WriteString(strings.Repeat(`\`, backslashes))
		backslashes = 0
		quoted.WriteRune(r)
	}
	quoted.WriteString(strings.Repeat(`\`, backslashes*2))
	quoted.WriteByte('"')

	value := escapeCmdMeta(quoted.String())
	if doubleMeta {
		value = escapeCmdMeta(value)
	}
	return value
}

// CmdJoin renders argv as one command line for cmd.exe. It is intended for terminal
// launchers which deliberately keep a command prompt open; ordinary subprocesses
// should use Command or CommandContext instead.
func CmdJoin(argv []string) (string, error) { return cmdJoin(argv, false) }

func cmdJoin(argv []string, doubleMeta bool) (string, error) {
	if len(argv) == 0 {
		return "", nil
	}
	parts := make([]string, len(argv))
	for i, arg := range argv {
		if strings.ContainsAny(arg, "\r\n") {
			return "", fmt.Errorf("cmd.exe argument %d contains a newline", i)
		}
		if i == 0 {
			parts[i] = escapeCmdMeta(arg)
		} else {
			parts[i] = cmdQuote(arg, doubleMeta)
		}
	}
	return strings.Join(parts, " "), nil
}

func escapeCmdMeta(s string) string {
	var escaped strings.Builder
	escaped.Grow(len(s))
	for _, r := range s {
		if strings.ContainsRune(cmdMeta, r) {
			escaped.WriteByte('^')
		}
		escaped.WriteRune(r)
	}
	return escaped.String()
}
