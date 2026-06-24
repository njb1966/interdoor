package engine

import (
	"encoding/json"
	"fmt"
	"time"
)

// Obligation is the favor/debt record — the network's economic centerpiece
// (DATA_MODEL.md §1.3). One entity, two directions: it is a *favor* to whoever
// is the creditor and a *debt* to whoever is the debtor. Engine-generic and
// globally identified so it is cross-node-ready, even though v1 obligations are
// NPC-only and single-node.
type Obligation struct {
	ID         string
	SourceNode string
	Creditor   string // global player ID or "npc:<id>"
	Debtor     string
	Kind       string // favor | debt | contract
	Terms      string
	Weight     int
	Status     string // open | resolved
	CreatedAt  time.Time
}

// CreateObligation opens a new obligation and emits debt.created.
func (s *Store) CreateObligation(creditor, debtor, kind, terms string, weight int) (*Obligation, error) {
	s.mu.Lock()
	s.oseq++
	id := fmt.Sprintf("%s:o_%d", s.nodeID, s.oseq)
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO obligations(obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,created_at)
		 VALUES(?,?,?,?,?,?,?, 'open', ?)`,
		id, s.nodeID, creditor, debtor, kind, terms, weight, now.Unix())
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if err := s.Emit("debt.created", map[string]any{
		"obligation_id": id, "source_node": s.nodeID,
		"creditor_ref": creditor, "debtor_ref": debtor,
		"kind": kind, "terms": terms, "weight": weight,
	}); err != nil {
		return nil, err
	}
	return &Obligation{ID: id, SourceNode: s.nodeID, Creditor: creditor, Debtor: debtor,
		Kind: kind, Terms: terms, Weight: weight, Status: "open", CreatedAt: now}, nil
}

// DebtLoad sums a debtor's open obligation weights (DATA_MODEL.md §1.2 derived).
func (s *Store) DebtLoad(debtor string) (int, error) {
	var sum int
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(weight),0) FROM obligations WHERE debtor_ref=? AND status='open'`, debtor).
		Scan(&sum)
	return sum, err
}

// OpenDebts lists a debtor's open obligations, oldest first.
func (s *Store) OpenDebts(debtor string) ([]Obligation, error) {
	rows, err := s.db.Query(
		`SELECT obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,created_at
		 FROM obligations WHERE debtor_ref=? AND status='open' ORDER BY created_at ASC, obligation_id ASC`, debtor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Obligation
	for rows.Next() {
		var o Obligation
		var created int64
		if err := rows.Scan(&o.ID, &o.SourceNode, &o.Creditor, &o.Debtor, &o.Kind,
			&o.Terms, &o.Weight, &o.Status, &created); err != nil {
			return nil, err
		}
		o.CreatedAt = time.Unix(created, 0)
		out = append(out, o)
	}
	return out, rows.Err()
}

// PayDebt applies amount against a debtor's open obligations, oldest first.
// Fully covered obligations are resolved (emitting debt.resolved); a partially
// covered one has its weight reduced and emits debt.adjusted. Returns the amount
// actually applied.
func (s *Store) PayDebt(debtor string, amount int) (int, error) {
	debts, err := s.OpenDebts(debtor)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, o := range debts {
		if amount <= 0 {
			break
		}
		if amount >= o.Weight {
			if err := s.resolve(o.ID, "paid"); err != nil {
				return applied, err
			}
			amount -= o.Weight
			applied += o.Weight
		} else {
			newWeight := o.Weight - amount
			s.mu.Lock()
			_, err := s.db.Exec(`UPDATE obligations SET weight=weight-? WHERE obligation_id=?`, amount, o.ID)
			s.mu.Unlock()
			if err != nil {
				return applied, err
			}
			if err := s.Emit("debt.adjusted", map[string]any{
				"obligation_id": o.ID,
				"old_weight":    o.Weight,
				"new_weight":    newWeight,
				"delta":         -amount,
				"reason":        "partial_payment",
			}); err != nil {
				return applied, err
			}
			applied += amount
			amount = 0
		}
	}
	return applied, nil
}

// RegisterDebtHandlers wires the OnEvent callbacks that apply foreign debt events
// from other nodes into this node's local obligations table. Call once at startup,
// after Migrate(). Idempotency is guaranteed by ApplyEvent (handlers run once per
// event_id and handler index); INSERT OR IGNORE handles any edge-case duplicates
// at the DB level.
func (s *Store) RegisterDebtHandlers() {
	s.OnEvent("debt.created", func(e Event) error {
		var p struct {
			ObligationID string `json:"obligation_id"`
			SourceNode   string `json:"source_node"`
			CreditorRef  string `json:"creditor_ref"`
			DebtorRef    string `json:"debtor_ref"`
			Kind         string `json:"kind"`
			Terms        string `json:"terms"`
			Weight       int    `json:"weight"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := s.db.Exec(
			`INSERT OR IGNORE INTO obligations
			 (obligation_id,source_node,creditor_ref,debtor_ref,kind,terms,weight,status,created_at)
			 VALUES(?,?,?,?,?,?,?,'open',?)`,
			p.ObligationID, p.SourceNode, p.CreditorRef, p.DebtorRef,
			p.Kind, p.Terms, p.Weight, e.Timestamp.Unix())
		return err
	})

	s.OnEvent("debt.resolved", func(e Event) error {
		var p struct {
			ObligationID string `json:"obligation_id"`
			Resolution   string `json:"resolution"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := s.db.Exec(
			`UPDATE obligations SET status='resolved', resolved_at=?, resolution=?
			 WHERE obligation_id=? AND status='open'`,
			e.Timestamp.Unix(), p.Resolution, p.ObligationID)
		return err
	})

	s.OnEvent("debt.adjusted", func(e Event) error {
		var p struct {
			ObligationID string `json:"obligation_id"`
			NewWeight    int    `json:"new_weight"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		_, err := s.db.Exec(
			`UPDATE obligations SET weight=? WHERE obligation_id=? AND status='open'`,
			p.NewWeight, p.ObligationID)
		return err
	})
}

func (s *Store) resolve(id, resolution string) error {
	resolvedAt := time.Now().Unix()
	s.mu.Lock()
	_, err := s.db.Exec(
		`UPDATE obligations SET status='resolved', resolved_at=?, resolution=? WHERE obligation_id=?`,
		resolvedAt, resolution, id)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return s.Emit("debt.resolved", map[string]any{
		"obligation_id": id,
		"resolution":    resolution,
		"resolved_at":   resolvedAt,
	})
}
