package engine

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "t.db"), "node01")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestTurnEconomy(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateAccount("Tester", "secret")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	turn, _ := s.LoadTurn(p.GlobalID)
	if turn.Actions != MainActionsPerDay {
		t.Fatalf("start actions: want %d, got %d", MainActionsPerDay, turn.Actions)
	}
	if turn.Attacks != AttacksPerDay {
		t.Fatalf("start attacks: want %d, got %d", AttacksPerDay, turn.Attacks)
	}
	if err := s.SpendActions(p.GlobalID, 2); err != nil {
		t.Fatalf("spend: %v", err)
	}
	turn, _ = s.LoadTurn(p.GlobalID)
	if turn.Actions != MainActionsPerDay-2 {
		t.Fatalf("after spend: want %d, got %d", MainActionsPerDay-2, turn.Actions)
	}
	if err := s.SpendActions(p.GlobalID, 100); err == nil {
		t.Fatalf("overspend should fail")
	}
}

func TestDailyResetNoRollover(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateAccount("Tester", "secret")
	_ = s.SpendActions(p.GlobalID, 5) // 7 left

	yesterday := time.Now().AddDate(0, 0, -1).Unix()
	if _, err := s.DB().Exec(`UPDATE turn_state SET last_reset=? WHERE global_id=?`, yesterday, p.GlobalID); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	reset, err := s.ResetIfNewDay(p.GlobalID)
	if err != nil || !reset {
		t.Fatalf("expected reset, got reset=%v err=%v", reset, err)
	}
	turn, _ := s.LoadTurn(p.GlobalID)
	if turn.Actions != MainActionsPerDay {
		t.Fatalf("no rollover: want %d, got %d", MainActionsPerDay, turn.Actions)
	}
	if reset, _ := s.ResetIfNewDay(p.GlobalID); reset {
		t.Fatalf("should not reset twice the same day")
	}
}
