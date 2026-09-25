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
		// The read can cut a multi-byte rune; drop up to UTFMax-1 undecodable trailing bytes.
		for i := 0; i < utf8.UTFMax-1 && len(buf) > 0; i++ {
			if r, size := utf8.DecodeLastRune(buf); r != utf8.RuneError || size != 1 {
				break
			}
			buf = buf[:len(buf)-1]
		}
	}
	return utf8.Valid(buf)
}
