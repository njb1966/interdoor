package engine

// Node-local federation sync cursors (FEDERATION_PROTOCOL.md §14). All Local
// state — never leaves the node. `hub_cursor` is the last applied feed position;
// `last_pushed_seq` is the highest local event seq delivered to the hub; `api_key`
// is this node's hub credential.

// SyncState returns the node's sync cursors and hub API key.
func (s *Store) SyncState() (hubCursor, lastPushed int64, apiKey string, err error) {
	err = s.db.QueryRow(`SELECT hub_cursor,last_pushed_seq,api_key FROM sync_state WHERE id=1`).
		Scan(&hubCursor, &lastPushed, &apiKey)
	return
}

// SetHubCursor records the last feed position applied from the hub.
func (s *Store) SetHubCursor(c int64) error { return s.setSync("hub_cursor", c) }

// SetPushCursor records the highest local event seq confirmed delivered.
func (s *Store) SetPushCursor(c int64) error { return s.setSync("last_pushed_seq", c) }

func (s *Store) setSync(col string, v int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// col is an internal constant, never user input.
	_, err := s.db.Exec(`UPDATE sync_state SET `+col+`=? WHERE id=1`, v)
	return err
}

// SetAPIKey stores this node's hub credential.
func (s *Store) SetAPIKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE sync_state SET api_key=? WHERE id=1`, key)
	return err
}
