package engine

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestEventsSinceOrdersAndFilters(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 3; i++ {
		if err := s.Emit("test.tick", map[string]int{"n": i}); err != nil {
			t.Fatalf("emit: %v", err)
		}
	}
	all, err := s.EventsSince("node01", 0)
	if err != nil {
		t.Fatalf("since: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("want 3 events, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i].Seq <= all[i-1].Seq {
			t.Fatalf("events not in seq order")
		}
	}
	if tail, _ := s.EventsSince("node01", all[0].Seq); len(tail) != 2 {
		t.Fatalf("after first seq: want 2, got %d", len(tail))
	}
}

func TestApplyEventRetriesFailedHandler(t *testing.T) {
	s := newTestStore(t)
	calls := 0
	s.OnEvent("debt.created", func(Event) error {
		calls++
		if calls == 1 {
			return errors.New("transient handler failure")
		}
		return nil
	})

	e := Event{
		EventID: "nodeB:8", SourceNode: "nodeB", Seq: 8, Type: "debt.created",
		Timestamp: time.Now(), Payload: json.RawMessage(`{"weight":5}`),
	}
	if newly, err := s.ApplyEvent(e); err == nil || !newly {
		t.Fatalf("first apply should store event and fail handler: newly=%v err=%v", newly, err)
	}
	if newly, err := s.ApplyEvent(e); err != nil || newly {
		t.Fatalf("replay should retry handler without duplicate event: newly=%v err=%v", newly, err)
	}
	if newly, err := s.ApplyEvent(e); err != nil || newly {
		t.Fatalf("post-success replay should be no-op: newly=%v err=%v", newly, err)
	}
	if calls != 2 {
		t.Fatalf("handler calls: want 2, got %d", calls)
	}
	if got, _ := s.EventsSince("nodeB", 0); len(got) != 1 {
		t.Fatalf("event row should not duplicate, got %d", len(got))
	}
}

func TestApplyEventRetriesFailedHandlerAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.db")
	s, err := Open(path, "node01")
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate first store: %v", err)
	}
	s.OnEvent("debt.created", func(Event) error {
		return errors.New("transient handler failure")
	})
	e := Event{
		EventID: "nodeB:9", SourceNode: "nodeB", Seq: 9, Type: "debt.created",
		Timestamp: time.Now(), Payload: json.RawMessage(`{"weight":5}`),
	}
	if newly, err := s.ApplyEvent(e); err == nil || !newly {
		t.Fatalf("first apply should store event and fail handler: newly=%v err=%v", newly, err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	reopened, err := Open(path, "node01")
	if err != nil {
		t.Fatalf("open restarted store: %v", err)
	}
	if err := reopened.Migrate(); err != nil {
		t.Fatalf("migrate restarted store: %v", err)
	}
	defer reopened.Close()
	calls := 0
	reopened.OnEvent("debt.created", func(Event) error {
		calls++
		return nil
	})
	if newly, err := reopened.ApplyEvent(e); err != nil || newly {
		t.Fatalf("restart replay should retry handler without duplicate event: newly=%v err=%v", newly, err)
	}
	if calls != 1 {
		t.Fatalf("handler calls after restart: want 1, got %d", calls)
	}
	var status string
	if err := reopened.DB().QueryRow(
		`SELECT status FROM event_handler_state WHERE event_id=? AND handler_index=0`,
		e.EventID,
	).Scan(&status); err != nil {
		t.Fatalf("handler state query: %v", err)
	}
	if status != "applied" {
		t.Fatalf("handler status after restart replay: want applied, got %q", status)
	}
	if got, _ := reopened.EventsSince("nodeB", 0); len(got) != 1 {
		t.Fatalf("event row should not duplicate after restart, got %d", len(got))
	}
}

func TestApplyEventIdempotent(t *testing.T) {
	s := newTestStore(t)
	calls := 0
	s.OnEvent("debt.created", func(Event) error { calls++; return nil })

	e := Event{
		EventID: "nodeB:7", SourceNode: "nodeB", Seq: 7, Type: "debt.created",
		Timestamp: time.Now(), Payload: json.RawMessage(`{"weight":5}`),
	}
	if newly, err := s.ApplyEvent(e); err != nil || !newly {
		t.Fatalf("first apply should be new: newly=%v err=%v", newly, err)
	}
	if newly, err := s.ApplyEvent(e); err != nil || newly {
		t.Fatalf("replay should be a duplicate no-op: newly=%v err=%v", newly, err)
	}
	if calls != 1 {
		t.Fatalf("handler must run exactly once, ran %d", calls)
	}
	if got, _ := s.EventsSince("nodeB", 0); len(got) != 1 || got[0].EventID != "nodeB:7" {
		t.Fatalf("foreign event should be stored and readable, got %+v", got)
	}
}
