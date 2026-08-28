// Package textfile answers one question about a path: does it hold text, or bytes no
// editor should be pointed at?
//
// It exists because the sniff was written for gote's document scan and then wanted by
// gofer's file menu, which cannot reach into another module's internal package. The test
// is content-based rather than extension-based deliberately: that is what lets a caller
// list or act on files with no extension at all (Makefile, LICENSE, .gitignore) without
// maintaining a name table that is wrong the first time someone invents a suffix.
package textfile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"unicode/utf8"
)

// sniffLen is how much of a file's head the text test reads. Enough to catch every
// binary format's magic bytes and any realistic UTF-8 breakage, small enough that a
// recursive scan pays one page read per candidate.
const sniffLen = 512

// IsText reports whether path holds text, judged from its first sniffLen bytes: a NUL
// byte or invalid UTF-8 means binary. An unreadable path (including a symlink pointing at
// a directory, and a directory itself) is not text.
//
// An empty file is text. That is not a nicety: a caller that creates a file and then
// re-scans to list it would watch every newly created document vanish if zero bytes read
// as binary.
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
		// The read boundary can cut a multi-byte rune in half; that is our truncation,
		// not the file's corruption. Drop up to UTFMax-1 trailing bytes that don't
		// decode, then judge what's left.
		for i := 0; i < utf8.UTFMax-1 && len(buf) > 0; i++ {
			if r, size := utf8.DecodeLastRune(buf); r != utf8.RuneError || size != 1 {
				break
			}
			buf = buf[:len(buf)-1]
		}
	}
	return utf8.Valid(buf)
}
