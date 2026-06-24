package game

import (
	"math/rand"
	"path/filepath"
	"testing"

	"interdoor.net/interdoor/internal/engine"
)

func TestTransferLootConservesGoods(t *testing.T) {
	s, err := engine.Open(filepath.Join(t.TempDir(), "t.db"), "node01")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		t.Fatalf("engine migrate: %v", err)
	}
	g := New("node01")
	if err := g.Migrate(s.DB()); err != nil {
		t.Fatalf("game migrate: %v", err)
	}
	victim, _ := s.CreateAccount("Victim", "secret")
	attacker, _ := s.CreateAccount("Attacker", "secret")
	_ = g.NewCharacter(s.DB(), victim)
	_ = g.NewCharacter(s.DB(), attacker)

	const qty = 10
	if err := addGoods(s.DB(), victim.GlobalID, "trd_brass_fittings", qty); err != nil {
		t.Fatalf("seed goods: %v", err)
	}
	total := qty * goodWeight("trd_brass_fittings")

	rng := rand.New(rand.NewSource(1))
	desc, items := transferLoot(s.DB(), victim.GlobalID, attacker.GlobalID, rng)
	if len(items) == 0 {
		t.Fatalf("expected loot to be taken, got %q", desc)
	}
	gotA := goodsValue(s.DB(), attacker.GlobalID)
	gotV := goodsValue(s.DB(), victim.GlobalID)
	if gotA == 0 {
		t.Fatalf("attacker received nothing")
	}
	if gotV == 0 {
		t.Fatalf("victim should keep the majority (25-50%% taken)")
	}
	if gotA+gotV != total {
		t.Fatalf("goods not conserved: attacker %d + victim %d != %d", gotA, gotV, total)
	}
}

// A winning attacker must take at least one good even when the victim carries a
// single low-value item (25-50% of 1 would otherwise round to nothing).
func TestTransferLootGuaranteesOne(t *testing.T) {
	s, err := engine.Open(filepath.Join(t.TempDir(), "t.db"), "node01")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	_ = s.Migrate()
	g := New("node01")
	_ = g.Migrate(s.DB())
	victim, _ := s.CreateAccount("Victim", "secret")
	attacker, _ := s.CreateAccount("Attacker", "secret")
	_ = g.NewCharacter(s.DB(), victim)
	_ = g.NewCharacter(s.DB(), attacker)
	_ = addGoods(s.DB(), victim.GlobalID, "trd_bone_buttons", 1)

	rng := rand.New(rand.NewSource(1))
	_, items := transferLoot(s.DB(), victim.GlobalID, attacker.GlobalID, rng)
	if len(items) == 0 {
		t.Fatalf("a win must take at least one good")
	}
	if goodsValue(s.DB(), attacker.GlobalID) == 0 {
		t.Fatalf("attacker received nothing")
	}
}
