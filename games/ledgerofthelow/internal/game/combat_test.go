package game

import (
	"math/rand"
	"testing"
)

func TestDamageNeverBelowOne(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		if d, _ := Damage(10, 4, 5, r); d < 1 {
			t.Fatalf("damage below 1: %d", d)
		}
	}
	// Wildly out-defended attacker still lands a chip hit.
	for i := 0; i < 200; i++ {
		if d, _ := Damage(1, 100, 0, r); d < 1 {
			t.Fatalf("min damage below 1: %d", d)
		}
	}
}

func TestCritCertainty(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	if _, crit := Damage(10, 0, 100, r); !crit {
		t.Fatalf("luck 100 must always crit")
	}
	if _, crit := Damage(10, 0, 0, r); crit {
		t.Fatalf("luck 0 must never crit")
	}
}

func TestFleeChance(t *testing.T) {
	if fleeChance(0) != 40 {
		t.Fatalf("base flee: want 40, got %d", fleeChance(0))
	}
	if fleeChance(5) != 50 {
		t.Fatalf("flee with luck 5: want 50, got %d", fleeChance(5))
	}
}
