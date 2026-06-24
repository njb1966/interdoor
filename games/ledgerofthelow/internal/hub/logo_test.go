package hub

import "testing"

func TestLogoLinesFitPortalHeader(t *testing.T) {
	if len(logoLines) != 9 {
		t.Fatalf("logoLines length = %d, want 9 to preserve portal layout", len(logoLines))
	}
	for i, line := range logoLines {
		if got := ansiVisibleLen(line); got > 80 {
			t.Fatalf("logo line %d is %d columns, want <= 80", i+1, got)
		}
	}
}

func ansiVisibleLen(s string) int {
	out := 0
	rs := []rune(s)
	for i := 0; i < len(rs); {
		if rs[i] == '\x1b' && i+1 < len(rs) && rs[i+1] == '[' {
			i += 2
			for i < len(rs) && (rs[i] < '@' || rs[i] > '~') {
				i++
			}
			if i < len(rs) {
				i++
			}
			continue
		}
		out++
		i++
	}
	return out
}
