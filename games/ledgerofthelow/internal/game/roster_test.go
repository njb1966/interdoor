package game

import (
	"strings"
	"testing"
	"time"

	"interdoor.net/interdoor/internal/engine/term"
)

func TestRemoteRosterNoteMarksStaleEntries(t *testing.T) {
	now := time.Unix(1000, 0)
	note := remoteRosterNote("node01", "node02", now.Add(-16*time.Minute), now)
	note = strings.ReplaceAll(note, term.FG(term.Cyan), "")
	note = strings.ReplaceAll(note, term.Reset(), "")

	for _, want := range []string{"from node02", "stale", "seen 16m ago"} {
		if !strings.Contains(note, want) {
			t.Fatalf("remote roster note missing %q: %q", want, note)
		}
	}
}

func TestRemoteRosterNoteKeepsFreshEntriesCurrent(t *testing.T) {
	now := time.Unix(1000, 0)
	note := remoteRosterNote("node01", "node02", now.Add(-5*time.Minute), now)

	if strings.Contains(note, "stale") {
		t.Fatalf("fresh remote roster note should not be stale: %q", note)
	}
	if !strings.Contains(note, "seen 5m ago") {
		t.Fatalf("fresh remote roster note should include last-seen age: %q", note)
	}
	if local := remoteRosterNote("node01", "node01", now.Add(-time.Hour), now); local != "" {
		t.Fatalf("local roster note should be empty, got %q", local)
	}
}
