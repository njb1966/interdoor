package engine

import "testing"

func TestSpendAttackBudget(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateAccount("Aggressor", "secret")
	turn, _ := s.LoadTurn(p.GlobalID)
	if turn.Attacks != AttacksPerDay {
		t.Fatalf("start attacks: want %d, got %d", AttacksPerDay, turn.Attacks)
	}
	for i := 0; i < AttacksPerDay; i++ {
		if err := s.SpendAttack(p.GlobalID); err != nil {
			t.Fatalf("spend %d: %v", i, err)
		}
	}
	if err := s.SpendAttack(p.GlobalID); err == nil {
		t.Fatalf("spending past the budget should fail")
	}
}

func TestPlayersExcludesSelf(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.CreateAccount("Aaa", "secret")
	b, _ := s.CreateAccount("Bbb", "secret")
	others, err := s.Players(a.GlobalID)
	if err != nil {
		t.Fatalf("players: %v", err)
	}
	if len(others) != 1 || others[0].GlobalID != b.GlobalID {
		t.Fatalf("roster should be exactly [Bbb], got %+v", others)
	}
}
