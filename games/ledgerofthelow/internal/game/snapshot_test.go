package game

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"interdoor.net/interdoor/internal/engine"
)

func TestSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Home node: create a character and give it some distinctive state.
	home, err := engine.Open(filepath.Join(dir, "home.db"), "node01")
	if err != nil {
		t.Fatalf("open home: %v", err)
	}
	defer home.Close()
	_ = home.Migrate()
	g1 := New("node01")
	_ = g1.Migrate(home.DB())

	p, err := home.CreateAccount("Traveler", "passphrase")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = g1.NewCharacter(home.DB(), p)
	_ = addGoods(home.DB(), p.GlobalID, "trd_brass_fittings", 4)
	if _, err := home.DB().Exec(
		`UPDATE char_state SET weapon='wpn_dock_hook', depth_record=3, blooded=1, hp=15 WHERE global_id=?`,
		p.GlobalID); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	// Export, then round-trip through JSON (the wire format).
	snap, err := home.ExportPlayer(p.GlobalID, g1)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	wire, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var arrived engine.Snapshot
	if err := json.Unmarshal(wire, &arrived); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Visiting node (fresh, different node id): import the snapshot.
	away, err := engine.Open(filepath.Join(dir, "away.db"), "node02")
	if err != nil {
		t.Fatalf("open away: %v", err)
	}
	defer away.Close()
	_ = away.Migrate()
	g2 := New("node02")
	_ = g2.Migrate(away.DB())
	if err := away.ImportPlayer(&arrived, g2); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Credential portability: the same password authenticates on the visiting node.
	visiting, err := away.Authenticate("Traveler", "passphrase")
	if err != nil {
		t.Fatalf("auth on visiting node failed: %v", err)
	}
	// Ownership stays with the home node.
	if visiting.HomeNode != "node01" {
		t.Fatalf("home node not preserved: %q", visiting.HomeNode)
	}
	if visiting.GlobalID != p.GlobalID {
		t.Fatalf("global id changed: %q != %q", visiting.GlobalID, p.GlobalID)
	}

	// Character state reconstructed exactly.
	c, err := loadChar(away.DB(), p.GlobalID)
	if err != nil {
		t.Fatalf("load char on away: %v", err)
	}
	if c.Weapon != "wpn_dock_hook" || c.DepthRecord != 3 || !c.Blooded || c.HP != 15 {
		t.Fatalf("character not reconstructed: %+v", c)
	}
	if got, want := goodsValue(away.DB(), p.GlobalID), 4*goodWeight("trd_brass_fittings"); got != want {
		t.Fatalf("goods not transferred: got %d, want %d", got, want)
	}
	// The visiting player has a local turn budget to play with.
	if turn, err := away.LoadTurn(p.GlobalID); err != nil || turn.Actions != engine.MainActionsPerDay {
		t.Fatalf("visiting turn budget: %+v err=%v", turn, err)
	}
}
