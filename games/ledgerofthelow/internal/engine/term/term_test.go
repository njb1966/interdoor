package term

import (
	"strings"
	"testing"
)

func TestClipTruncatesToWidth(t *testing.T) {
	out := clip(strings.Repeat("x", 200))
	if got := len([]rune(out)); got != Width {
		t.Fatalf("clip width: want %d, got %d", Width, got)
	}
}

func TestClipPassesShortLines(t *testing.T) {
	in := "  HP 18/20"
	if clip(in) != in {
		t.Fatalf("short line should be unchanged")
	}
}

func TestClipPassesEscapeLines(t *testing.T) {
	in := "\x1b[31m" + strings.Repeat("y", 100)
	if clip(in) != in {
		t.Fatalf("escape-bearing line should pass through untouched")
	}
}

func TestWrapLineRespectsWidthAndIndent(t *testing.T) {
	in := "  Something long unknots itself from the black water, decides you are food, and is only partly wrong."
	out := wrapLine(in, Width)
	if len(out) < 2 {
		t.Fatalf("expected the long line to wrap, got %d line(s)", len(out))
	}
	for i, ln := range out {
		if len([]rune(ln)) > Width {
			t.Fatalf("line %d exceeds width: %d", i, len([]rune(ln)))
		}
	}
	for _, ln := range out[1:] {
		if !strings.HasPrefix(ln, "  ") {
			t.Fatalf("continuation line lost its indent: %q", ln)
		}
	}
}

func TestWrapShortLineUnchanged(t *testing.T) {
	in := "  HP 20/20"
	out := wrapLine(in, Width)
	if len(out) != 1 || out[0] != in {
		t.Fatalf("short line should pass through as one line")
	}
}
