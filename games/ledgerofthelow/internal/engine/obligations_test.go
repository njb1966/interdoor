package engine

import "testing"

func TestObligationLifecycle(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateAccount("Debtor", "secret")

	if d, _ := s.DebtLoad(p.GlobalID); d != 0 {
		t.Fatalf("start debt: want 0, got %d", d)
	}

	o, err := s.CreateObligation("npc:npc_maren", p.GlobalID, "debt", "vault plate advanced", 30)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if o.ID != "node01:o_1" {
		t.Fatalf("obligation id format: want node01:o_1, got %q", o.ID)
	}
	if d, _ := s.DebtLoad(p.GlobalID); d != 30 {
		t.Fatalf("after create: want 30, got %d", d)
	}

	// Partial payment reduces weight, no resolution yet.
	applied, err := s.PayDebt(p.GlobalID, 10)
	if err != nil || applied != 10 {
		t.Fatalf("partial pay: applied=%d err=%v", applied, err)
	}
	if d, _ := s.DebtLoad(p.GlobalID); d != 20 {
		t.Fatalf("after partial: want 20, got %d", d)
	}

	// Overpayment clears the remainder (applies only what's owed).
	applied, _ = s.PayDebt(p.GlobalID, 100)
	if applied != 20 {
		t.Fatalf("final pay: want 20 applied, got %d", applied)
	}
	if d, _ := s.DebtLoad(p.GlobalID); d != 0 {
		t.Fatalf("after full pay: want 0, got %d", d)
	}

	var created, resolved int
	_ = s.DB().QueryRow(`SELECT COUNT(*) FROM events WHERE type='debt.created'`).Scan(&created)
	_ = s.DB().QueryRow(`SELECT COUNT(*) FROM events WHERE type='debt.resolved'`).Scan(&resolved)
	var adjusted int
	_ = s.DB().QueryRow(`SELECT COUNT(*) FROM events WHERE type='debt.adjusted'`).Scan(&adjusted)
	if created != 1 {
		t.Fatalf("debt.created events: want 1, got %d", created)
	}
	if adjusted != 1 {
		t.Fatalf("debt.adjusted events: want 1, got %d", adjusted)
	}
	if resolved != 1 {
		t.Fatalf("debt.resolved events: want 1, got %d", resolved)
	}
}

func TestDebtFIFO(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateAccount("Owing", "secret")
	_, _ = s.CreateObligation("npc:npc_maren", p.GlobalID, "debt", "first", 10)
	_, _ = s.CreateObligation("npc:npc_maren", p.GlobalID, "debt", "second", 10)
	if d, _ := s.DebtLoad(p.GlobalID); d != 20 {
		t.Fatalf("two debts: want 20, got %d", d)
	}
	// Pay exactly the first.
	if applied, _ := s.PayDebt(p.GlobalID, 10); applied != 10 {
		t.Fatalf("pay first: want 10, got %d", applied)
	}
	open, _ := s.OpenDebts(p.GlobalID)
	if len(open) != 1 || open[0].Terms != "second" {
		t.Fatalf("FIFO: expected only 'second' open, got %+v", open)
	}
}
