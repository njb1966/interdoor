// Package fed is the node-side federation client (FEDERATION_PROTOCOL.md §14):
// it registers with the hub and runs the push/pull event-sync loop. It is
// engine-generic — any game on InterDOOR reuses it unchanged. It talks to the hub
// only over the JSON wire protocol (no shared Go types with the hub package).
package fed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"interdoor.net/interdoor/internal/engine"
)

// ProtocolVersion is the wire-protocol version the node speaks.
const ProtocolVersion = "1"

// Client is an HTTP client for one hub.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type wireEvent struct {
	EventID    string          `json:"event_id"`
	SourceNode string          `json:"source_node"`
	Seq        int64           `json:"seq"`
	Type       string          `json:"type"`
	TS         int64           `json:"ts"`
	Payload    json.RawMessage `json:"payload"`
}

type feedEvent struct {
	HubSeq int64 `json:"hub_seq"`
	wireEvent
}

// Pulled is a feed event with its hub position.
type Pulled struct {
	HubSeq int64
	Event  engine.Event
}

// Register joins the network and returns the issued API key. gameTitle is optional
// display metadata; gameID remains the stable compatibility identifier.
func (c *Client) Register(regToken, nodeID, gameID, gameVersion, advertise string, gameTitle ...string) (string, error) {
	req := map[string]string{
		"node_id": nodeID, "registration_token": regToken, "game_id": gameID,
		"game_version": gameVersion, "protocol_version": ProtocolVersion, "advertise_addr": advertise,
	}
	if len(gameTitle) > 0 && gameTitle[0] != "" {
		req["game_title"] = gameTitle[0]
	}
	body, _ := json.Marshal(req)
	var resp struct {
		APIKey string `json:"api_key"`
	}
	if err := c.do(http.MethodPost, "/v1/register", body, &resp); err != nil {
		return "", err
	}
	return resp.APIKey, nil
}

// Heartbeat reports liveness and metrics.
func (c *Client) Heartbeat(nodeID string, players, uptimeS int, gameVersion string) error {
	body, _ := json.Marshal(map[string]any{
		"node_id": nodeID, "player_count": players, "uptime_s": uptimeS, "game_version": gameVersion,
	})
	return c.do(http.MethodPost, "/v1/heartbeat", body, nil)
}

// Push delivers locally-emitted events to the hub.
func (c *Client) Push(events []engine.Event) (accepted int, head int64, err error) {
	wires := make([]wireEvent, len(events))
	for i, e := range events {
		wires[i] = wireEvent{e.EventID, e.SourceNode, e.Seq, e.Type, e.Timestamp.Unix(), e.Payload}
	}
	body, _ := json.Marshal(map[string]any{"events": wires})
	var resp struct {
		Accepted   int   `json:"accepted"`
		LastHubSeq int64 `json:"last_hub_seq"`
	}
	if err := c.do(http.MethodPost, "/v1/events", body, &resp); err != nil {
		return 0, 0, err
	}
	return resp.Accepted, resp.LastHubSeq, nil
}

// Pull fetches feed events after a cursor. excludeSelf omits the node's own events.
func (c *Client) Pull(after int64, excludeSelf bool) ([]Pulled, int64, error) {
	path := fmt.Sprintf("/v1/events?after=%d&limit=500", after)
	if excludeSelf {
		path += "&exclude_self=true"
	}
	var resp struct {
		Head   int64       `json:"head"`
		Events []feedEvent `json:"events"`
	}
	if err := c.do(http.MethodGet, path, nil, &resp); err != nil {
		return nil, 0, err
	}
	out := make([]Pulled, len(resp.Events))
	for i, fe := range resp.Events {
		out[i] = Pulled{HubSeq: fe.HubSeq, Event: engine.Event{
			EventID: fe.EventID, SourceNode: fe.SourceNode, Seq: fe.Seq, Type: fe.Type,
			Timestamp: time.Unix(fe.TS, 0), Payload: fe.Payload,
		}}
	}
	return out, resp.Head, nil
}

type rosterEntry struct {
	GlobalID string `json:"global_id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Status   string `json:"status"`
	LastSeen int64  `json:"last_seen"`
}

type pvpPendingResp struct {
	Requests []struct {
		RequestID       string          `json:"request_id"`
		AttackerID      string          `json:"attacker_id"`
		VictimID        string          `json:"victim_id"`
		AttackerPayload json.RawMessage `json:"attacker"`
	} `json:"requests"`
}

// QueuePvP submits a cross-node attack to the hub. Returns the hub-assigned request ID.
func (c *Client) QueuePvP(attackerID, victimID string, payload json.RawMessage) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"attacker_id": attackerID, "victim_id": victimID, "attacker": payload,
	})
	var resp struct {
		RequestID string `json:"request_id"`
	}
	if err := c.do(http.MethodPost, "/v1/pvp", body, &resp); err != nil {
		return "", err
	}
	return resp.RequestID, nil
}

// PendingPvP fetches queued attacks targeting this node's players.
func (c *Client) PendingPvP() ([]engine.PvPPending, error) {
	var resp pvpPendingResp
	if err := c.do(http.MethodGet, "/v1/pvp/pending", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]engine.PvPPending, len(resp.Requests))
	for i, r := range resp.Requests {
		out[i] = engine.PvPPending{
			RequestID: r.RequestID, AttackerID: r.AttackerID,
			VictimID: r.VictimID, AttackerPayload: r.AttackerPayload,
		}
	}
	return out, nil
}

// CompletePvP reports a resolved attack back to the hub.
func (c *Client) CompletePvP(requestID string) error {
	return c.do(http.MethodPost, "/v1/pvp/"+requestID+"/result", []byte(`{}`), nil)
}

// BlockPvP marks a malformed queued attack blocked at the hub.
func (c *Client) BlockPvP(requestID, reason string) error {
	body, _ := json.Marshal(map[string]string{"error": reason})
	return c.do(http.MethodPost, "/v1/pvp/"+requestID+"/blocked", body, nil)
}

type travelSubmitWire struct {
	GlobalID string          `json:"global_id"`
	HomeNode string          `json:"home_node"`
	DestNode string          `json:"dest_node"`
	Snapshot json.RawMessage `json:"snapshot"`
}

type travelPendingResp struct {
	Arrivals []struct {
		TravelID string          `json:"travel_id"`
		GlobalID string          `json:"global_id"`
		HomeNode string          `json:"home_node"`
		FromNode string          `json:"from_node"`
		DestNode string          `json:"dest_node"`
		Snapshot json.RawMessage `json:"snapshot"`
	} `json:"arrivals"`
}

// SubmitTravel sends a player snapshot to the hub for relay to destNode.
func (c *Client) SubmitTravel(globalID, homeNode, destNode string, snapshot json.RawMessage) (string, error) {
	body, _ := json.Marshal(travelSubmitWire{GlobalID: globalID, HomeNode: homeNode, DestNode: destNode, Snapshot: snapshot})
	var resp struct {
		TravelID string `json:"travel_id"`
	}
	if err := c.do(http.MethodPost, "/v1/travel", body, &resp); err != nil {
		return "", err
	}
	return resp.TravelID, nil
}

// PendingTravel fetches player snapshots arriving at this node.
func (c *Client) PendingTravel() ([]engine.TravelPending, error) {
	var resp travelPendingResp
	if err := c.do(http.MethodGet, "/v1/travel/pending", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]engine.TravelPending, len(resp.Arrivals))
	for i, a := range resp.Arrivals {
		out[i] = engine.TravelPending{
			TravelID: a.TravelID, GlobalID: a.GlobalID,
			HomeNode: a.HomeNode, FromNode: a.FromNode,
			DestNode: a.DestNode, Snapshot: a.Snapshot,
		}
	}
	return out, nil
}

// ArriveTravel confirms that an arrival has been processed.
func (c *Client) ArriveTravel(travelID string) error {
	return c.do(http.MethodPost, "/v1/travel/"+travelID+"/arrived", []byte(`{}`), nil)
}

// BlockTravel marks a malformed queued arrival blocked at the hub.
func (c *Client) BlockTravel(travelID, reason string) error {
	body, _ := json.Marshal(map[string]string{"error": reason})
	return c.do(http.MethodPost, "/v1/travel/"+travelID+"/blocked", body, nil)
}

// PushRoster sends the node's current player list to the hub.
func (c *Client) PushRoster(players []engine.Player) error {
	entries := make([]rosterEntry, len(players))
	for i, p := range players {
		entries[i] = rosterEntry{
			GlobalID: p.GlobalID, Name: p.Name, Level: p.Level,
			Status: p.Status, LastSeen: p.LastSeen.Unix(),
		}
	}
	body, _ := json.Marshal(map[string]any{"entries": entries})
	return c.do(http.MethodPost, "/v1/roster", body, nil)
}

// PullRoster fetches all remote players from the hub, excluding this node's own.
func (c *Client) PullRoster() ([]engine.Player, error) {
	var resp struct {
		Entries []rosterEntry `json:"entries"`
	}
	if err := c.do(http.MethodGet, "/v1/roster?exclude_self=true", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]engine.Player, len(resp.Entries))
	for i, e := range resp.Entries {
		out[i] = engine.Player{
			GlobalID: e.GlobalID, HomeNode: e.NodeID, Name: e.Name,
			Level: e.Level, Status: e.Status, LastSeen: time.Unix(e.LastSeen, 0),
		}
	}
	return out, nil
}

func (c *Client) do(method, path string, body []byte, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	httpc := c.HTTP
	if httpc == nil {
		httpc = http.DefaultClient
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub %s %s: %d %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Syncer runs one node's federation loop: push local events, pull and apply
// foreign ones, advancing the node's cursors.
type Syncer struct {
	Store        *engine.Store
	Client       *Client
	NodeID       string
	GameVersion  string
	PvPResolve   engine.PvPResolveFn   // provided by game module; nil on standalone
	TravelImport engine.TravelImportFn // provided by node startup; nil on standalone
}

// Tick performs one push+pull cycle. Idempotent and crash-safe (cursors persist).
func (s *Syncer) Tick() error {
	players, _ := s.Store.LocalPlayers()
	if err := s.Client.Heartbeat(s.NodeID, len(players), 0, s.GameVersion); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	hubCursor, lastPushed, _, err := s.Store.SyncState()
	if err != nil {
		return err
	}
	// Push our own newly-emitted events.
	out, err := s.Store.EventsSince(s.NodeID, lastPushed)
	if err != nil {
		return err
	}
	if len(out) > 0 {
		if _, _, err := s.Client.Push(out); err != nil {
			return err
		}
		if err := s.Store.SetPushCursor(out[len(out)-1].Seq); err != nil {
			return err
		}
	}
	// Pull and apply everyone else's events (idempotent).
	pulled, _, err := s.Client.Pull(hubCursor, true)
	if err != nil {
		return err
	}
	for _, p := range pulled {
		if _, err := s.Store.ApplyEvent(p.Event); err != nil {
			return err
		}
		if err := s.Store.SetHubCursor(p.HubSeq); err != nil {
			return err
		}
	}
	// Drain pending cross-node PvP attacks targeting our players.
	if s.PvPResolve != nil {
		pending, err := s.Client.PendingPvP()
		if err != nil {
			return err
		}
		for _, req := range pending {
			if !rawObject(req.AttackerPayload) {
				reason := "malformed attacker payload"
				log.Printf("pvp block %s: %s", req.RequestID, reason)
				if err := s.Client.BlockPvP(req.RequestID, reason); err != nil {
					log.Printf("pvp block report %s: %v", req.RequestID, err)
				}
				continue
			}
			if err := s.PvPResolve(s.Store, req.RequestID, req.AttackerID, req.VictimID, req.AttackerPayload); err != nil {
				log.Printf("pvp resolve %s: %v", req.RequestID, err)
				continue
			}
			if err := s.Client.CompletePvP(req.RequestID); err != nil {
				log.Printf("pvp complete %s: %v", req.RequestID, err)
			}
		}
	}
	// Drain pending cross-node travel arrivals.
	if s.TravelImport != nil {
		arrivals, err := s.Client.PendingTravel()
		if err != nil {
			return err
		}
		for _, a := range arrivals {
			if !rawObject(a.Snapshot) {
				reason := "malformed travel snapshot"
				log.Printf("travel block %s: %s", a.TravelID, reason)
				if err := s.Client.BlockTravel(a.TravelID, reason); err != nil {
					log.Printf("travel block report %s: %v", a.TravelID, err)
				}
				continue
			}
			var snap engine.Snapshot
			if err := json.Unmarshal(a.Snapshot, &snap); err != nil {
				reason := "invalid travel snapshot: " + err.Error()
				log.Printf("travel block %s: %s", a.TravelID, reason)
				if err := s.Client.BlockTravel(a.TravelID, reason); err != nil {
					log.Printf("travel block report %s: %v", a.TravelID, err)
				}
				continue
			}
			if err := s.TravelImport(s.Store, &snap, a.FromNode, a.DestNode); err != nil {
				log.Printf("travel import %s: %v", a.TravelID, err)
				continue
			}
			if err := s.Client.ArriveTravel(a.TravelID); err != nil {
				log.Printf("travel arrive %s: %v", a.TravelID, err)
			}
		}
	}
	// Push only this node's own players — never remote_roster entries.
	locals, err := s.Store.LocalPlayers()
	if err != nil {
		return err
	}
	if err := s.Client.PushRoster(locals); err != nil {
		return err
	}
	// Pull and cache remote players so the wanderers screen shows cross-node visitors.
	remotes, err := s.Client.PullRoster()
	if err != nil {
		return err
	}
	return s.Store.UpsertRemoteRoster(remotes)
}

func rawObject(raw json.RawMessage) bool {
	var obj map[string]json.RawMessage
	if len(raw) == 0 {
		return false
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return obj != nil
}

// Run loops Tick on an interval until ctx is cancelled.
func (s *Syncer) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Tick(); err != nil {
				log.Printf("sync: %v", err)
			}
		}
	}
}
