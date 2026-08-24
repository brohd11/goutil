package strutil

import (
	"path/filepath"
	"testing"
)

// Depth backs depth-limited directory walks in three modules. Off by one here means a
// walk silently stops finding things one level early, so the boundary cases are the
// point: the base itself, a direct child, and a path Rel cannot express.
func TestDepth(t *testing.T) {
	base := filepath.Join("/tmp", "root")
	tests := []struct {
		name string
		path string
		want int
	}{
		{"base itself is zero", base, 0},
		{"direct child is one", filepath.Join(base, "a"), 1},
		{"grandchild is two", filepath.Join(base, "a", "b"), 2},
		{"deep", filepath.Join(base, "a", "b", "c", "d"), 4},
		{"a sibling counts the climb out", filepath.Join("/tmp", "other"), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Depth(base, tt.path); got != tt.want {
				t.Errorf("Depth(%q, %q) = %d, want %d", base, tt.path, got, tt.want)
			}
		})
	}
}

// An unrelated path -- one filepath.Rel cannot express as a relative walk -- reads as 0
// rather than leaking a negative or a garbage count into a caller's depth comparison.
func TestDepthUnrelatedPathIsZero(t *testing.T) {
	if got := Depth("relative/base", "/absolute/path"); got != 0 {
		t.Errorf("Depth across an unexpressible boundary = %d, want 0", got)
	}
}
