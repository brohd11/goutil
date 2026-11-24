// Package textfile reports whether a file holds text, judged by content rather than
// extension so files like Makefile and LICENSE count.
package textfile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"unicode/utf8"
)

// sniffLen is how much of a file's head IsText reads.
const sniffLen = 512

// IsText reports whether path holds text: no NUL byte and valid UTF-8 in its first
// sniffLen bytes. Unreadable paths and directories are not text; an empty file is.
func IsText(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false // unreadable, or a symlink pointing at a directory
	}
	defer f.Close()

	var b [sniffLen]byte
	n, err := f.Read(b[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	if n == 0 {
		return true // an empty file reads (0, io.EOF)
	}
	buf := b[:n]
	if bytes.IndexByte(buf, 0) >= 0 {
		return false
	}
	if n == sniffLen {
		// Only drop an incomplete rune that the read could have cut. FullRune treats
		// invalid encodings as complete, so they remain for Valid to reject below.
		start := len(buf) - 1
		for start > len(buf)-utf8.UTFMax && !utf8.RuneStart(buf[start]) {
			start--
		}
		if !utf8.FullRune(buf[start:]) {
			buf = buf[:start]
		}
	}
	return utf8.Valid(buf)
}
