package hub

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPortalDetailShowsExternalSSHInstruction(t *testing.T) {
	var out bytes.Buffer
	portalShowDetail(&out, Node{
		NodeID:        "thelow",
		GameID:        "ledger_of_the_low",
		GameTitle:     "Ledger of the Low",
		GameVersion:   "1.0.0",
		AdvertiseAddr: "ssh://node.interdoor.net:2323",
		PlayerCount:   1,
		LastHeartbeat: time.Now(),
	})
	got := out.String()
	for _, want := range []string{
		"This hub lists nodes; it does not launch game sessions.",
		"Open a new terminal and run:",
		"Ledger of the Low",
		"ssh -p 2323 node.interdoor.net",
		"Players:",
		"1 online",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("portal detail missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Connect with:") {
		t.Fatalf("portal detail still uses old ambiguous copy:\n%s", got)
	}
	if strings.Contains(got, "ledger_of_the_low") {
		t.Fatalf("portal detail shows protocol game_id instead of display title:\n%s", got)
	}
}
