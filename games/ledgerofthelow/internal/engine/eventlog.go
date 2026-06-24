package engine

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Event is a stored, self-contained record of something that happened
// (DATA_MODEL.md Part 3). Emit produces local events; ApplyEvent ingests foreign
// ones from other nodes. Ordering within a source is by Seq; identity is EventID
// (`source_node:seq`).
type Event struct {
	EventID    string
	SourceNode string
	Seq        int64
	Type       string
	Timestamp  time.Time
	Payload    json.RawMessage
}

// EventHandler reacts to a newly-applied event. Handlers run only the first time
// an event is applied (idempotency is the engine's responsibility, not theirs).
type EventHandler func(Event) error

// OnEvent registers a handler for an event type. The federation layer (B3) wires
// handlers that mutate local state from remote events; locally there are none yet.
func (s *Store) OnEvent(typ string, h EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[typ] = append(s.handlers[typ], h)
}

// EventsSince returns a source node's events with seq greater than afterSeq, in
// order. This is the outbound side of sync: what this node hands to others (and a
// handy inspection/replay primitive).
func (s *Store) EventsSince(sourceNode string, afterSeq int64) ([]Event, error) {
	rows, err := s.db.Query(
		`SELECT event_id,source_node,seq,type,ts,payload
		 FROM events WHERE source_node=? AND seq>? ORDER BY seq`, sourceNode, afterSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var ts int64
		var payload string
		if err := rows.Scan(&e.EventID, &e.SourceNode, &e.Seq, &e.Type, &ts, &payload); err != nil {
			return nil, err
		}
		e.Timestamp = time.Unix(ts, 0)
		e.Payload = json.RawMessage(payload)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ApplyEvent idempotently ingests an event (typically received from another node,
// per NETWORK_REQUIREMENTS Req 4). It returns true if the event row was newly
// stored, false if it was already present. Handler application is tracked
// separately so a handler failure can be retried on the next replay without
// inserting a duplicate event or rerunning handlers that already succeeded.
func (s *Store) ApplyEvent(e Event) (bool, error) {
	s.mu.Lock()
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO events(event_id,source_node,seq,type,ts,payload) VALUES(?,?,?,?,?,?)`,
		e.EventID, e.SourceNode, e.Seq, e.Type, e.Timestamp.Unix(), string(e.Payload))
	s.mu.Unlock()
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		for i, h := range s.handlersFor(e.Type) {
			applied, err := s.handlerApplied(e.EventID, i)
			if err != nil {
				return false, err
			}
			if applied {
				continue
			}
			if err := h(e); err != nil {
				_ = s.markHandler(e.EventID, i, "failed", err.Error())
				return false, err
			}
			if err := s.markHandler(e.EventID, i, "applied", ""); err != nil {
				return false, err
			}
		}
		return false, nil
	}
	for i, h := range s.handlersFor(e.Type) {
		if err := h(e); err != nil {
			_ = s.markHandler(e.EventID, i, "failed", err.Error())
			return true, err
		}
		if err := s.markHandler(e.EventID, i, "applied", ""); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (s *Store) handlerApplied(eventID string, index int) (bool, error) {
	var status string
	err := s.db.QueryRow(
		`SELECT status FROM event_handler_state WHERE event_id=? AND handler_index=?`,
		eventID, index).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return status == "applied", nil
}

func (s *Store) markHandler(eventID string, index int, status, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO event_handler_state(event_id,handler_index,status,error,updated_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(event_id, handler_index) DO UPDATE SET
		   status=excluded.status,
		   error=excluded.error,
		   updated_at=excluded.updated_at`,
		eventID, index, status, msg, time.Now().Unix())
	return err
}

func (s *Store) handlersFor(typ string) []EventHandler {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]EventHandler(nil), s.handlers[typ]...)
}
