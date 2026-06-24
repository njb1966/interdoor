package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Server is the hub's HTTP API (FEDERATION_PROTOCOL.md §4, §15).
type Server struct {
	store    Store
	regToken string // operator-set token a node must present to register
}

// NewServer builds the hub API over store. regToken gates registration.
func NewServer(store Store, regToken string) *Server {
	return &Server{store: store, regToken: regToken}
}

// Handler returns the routed HTTP handler (Go 1.22 method+path patterns).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleRoot)
	mux.HandleFunc("GET /v1/directory", s.handleDirectory)
	mux.HandleFunc("GET /v1/status", s.handleStatus)
	mux.HandleFunc("POST /v1/register", s.handleRegister)
	mux.HandleFunc("POST /v1/heartbeat", s.auth(s.handleHeartbeat))
	mux.HandleFunc("POST /v1/events", s.auth(s.handleEventsPush))
	mux.HandleFunc("GET /v1/events", s.auth(s.handleEventsPull))
	mux.HandleFunc("POST /v1/roster", s.auth(s.handleRosterPush))
	mux.HandleFunc("GET /v1/roster", s.auth(s.handleRosterPull))
	mux.HandleFunc("GET /v1/debts", s.auth(s.handleDebts))
	mux.HandleFunc("POST /v1/pvp", s.auth(s.handlePvPQueue))
	mux.HandleFunc("GET /v1/pvp/pending", s.auth(s.handlePvPPending))
	mux.HandleFunc("POST /v1/pvp/{id}/result", s.auth(s.handlePvPResult))
	mux.HandleFunc("POST /v1/pvp/{id}/blocked", s.auth(s.handlePvPBlocked))
	mux.HandleFunc("POST /v1/travel", s.auth(s.handleTravelSubmit))
	mux.HandleFunc("GET /v1/travel/pending", s.auth(s.handleTravelPending))
	mux.HandleFunc("POST /v1/travel/{id}/arrived", s.auth(s.handleTravelArrived))
	mux.HandleFunc("POST /v1/travel/{id}/blocked", s.auth(s.handleTravelBlocked))
	return mux
}

type registerReq struct {
	NodeID            string `json:"node_id"`
	RegistrationToken string `json:"registration_token"`
	GameID            string `json:"game_id"`
	GameTitle         string `json:"game_title"`
	GameVersion       string `json:"game_version"`
	ProtocolVersion   string `json:"protocol_version"`
	AdvertiseAddr     string `json:"advertise_addr"`
}

type registerResp struct {
	APIKey     string `json:"api_key"`
	HubSeqHead int64  `json:"hub_seq_head"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.RegistrationToken == "" || req.RegistrationToken != s.regToken {
		writeErr(w, http.StatusUnauthorized, "bad registration token")
		return
	}
	if req.ProtocolVersion != ProtocolVersion {
		writeErr(w, http.StatusUpgradeRequired, "unsupported protocol version")
		return
	}
	if req.NodeID == "" || req.GameID == "" {
		writeErr(w, http.StatusBadRequest, "node_id and game_id are required")
		return
	}
	gameTitle := strings.TrimSpace(req.GameTitle)
	if gameTitle == "" {
		gameTitle = req.GameID
	}
	key, hash := newAPIKey()
	n := Node{
		NodeID: req.NodeID, GameID: req.GameID, GameTitle: gameTitle, GameVersion: req.GameVersion,
		ProtocolVersion: req.ProtocolVersion, AdvertiseAddr: req.AdvertiseAddr,
	}
	if err := s.store.RegisterNode(n, hash); err != nil {
		if errors.Is(err, ErrNodeExists) {
			writeErr(w, http.StatusConflict, "node_id already registered")
			return
		}
		writeErr(w, http.StatusInternalServerError, "registration failed")
		return
	}
	writeJSON(w, http.StatusOK, registerResp{APIKey: key, HubSeqHead: 0})
}

type heartbeatReq struct {
	NodeID      string `json:"node_id"`
	PlayerCount int    `json:"player_count"`
	UptimeS     int    `json:"uptime_s"`
	GameVersion string `json:"game_version"`
}

type pendingCounts struct {
	Events int `json:"events"`
	PVP    int `json:"pvp"`
	Travel int `json:"travel"`
}

type heartbeatResp struct {
	HubSeqHead int64         `json:"hub_seq_head"`
	Pending    pendingCounts `json:"pending"`
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	var req heartbeatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.store.Heartbeat(n.NodeID, req.PlayerCount, req.UptimeS, req.GameVersion); err != nil {
		writeErr(w, http.StatusInternalServerError, "heartbeat failed")
		return
	}
	head, _ := s.store.FeedHead()
	pvpPending, _ := s.store.PendingPvPCount(n.NodeID)
	travelPending, _ := s.store.PendingTravelCount(n.NodeID)
	writeJSON(w, http.StatusOK, heartbeatResp{
		HubSeqHead: head,
		Pending:    pendingCounts{PVP: pvpPending, Travel: travelPending},
	})
}

// ---- event feed (§5) ----

type eventsPushReq struct {
	Events []WireEvent `json:"events"`
}

func (s *Server) handleEventsPush(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	var req eventsPushReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	// A node may only speak for itself (§5.1) — the spine of the trust model.
	for _, e := range req.Events {
		if e.SourceNode != n.NodeID {
			writeErr(w, http.StatusForbidden, "source_node must match the authenticated node")
			return
		}
	}
	if err := validateEventSchemas(req.Events); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.validateEventCompatibility(n, req.Events); err != nil {
		if errors.Is(err, ErrIncompatibleGame) {
			writeErr(w, http.StatusConflict, "event references incompatible game_id")
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "event references unknown node")
			return
		}
		writeErr(w, http.StatusInternalServerError, "event compatibility check failed")
		return
	}
	accepted, head, err := s.store.AppendEvents(req.Events)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "append failed")
		return
	}
	// Materialize debt.* events into the hub Ledger index (idempotent).
	_ = s.store.MaterializeEvents(req.Events)
	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": accepted, "duplicates": len(req.Events) - accepted, "last_hub_seq": head,
	})
}

func validateEventSchemas(events []WireEvent) error {
	for _, e := range events {
		if e.EventID == "" || e.SourceNode == "" || e.Type == "" || e.Seq < 1 || e.TS < 1 {
			return fmt.Errorf("invalid event envelope for %q", e.EventID)
		}
		fields, err := objectPayload(e)
		if err != nil {
			return err
		}
		switch e.Type {
		case "player.created":
			if err := requireStrings(fields, "global_id", "name", "home_node"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "created_at"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "player.died":
			if err := requireStrings(fields, "global_id"); err != nil {
				return eventSchemaErr(e, err)
			}
			if _, ok := fields["cause"]; !ok {
				return eventSchemaErr(e, errors.New("missing cause"))
			}
			if err := requireNumbers(fields, "timestamp"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "player.traveled":
			if err := requireStrings(fields, "global_id", "src_node", "dest_node", "snapshot_hash"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "timestamp"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "debt.created":
			if err := requireStrings(fields, "obligation_id", "creditor_ref", "debtor_ref", "kind", "terms"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "weight"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "debt.adjusted":
			if err := requireStrings(fields, "obligation_id", "reason"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "old_weight", "new_weight", "delta"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "debt.resolved":
			if err := requireStrings(fields, "obligation_id", "resolution"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "resolved_at"); err != nil {
				return eventSchemaErr(e, err)
			}
		case "pvp.resolved":
			if err := requireStrings(fields, "request_id", "attacker_global_id", "victim_global_id", "winner_global_id", "result_text"); err != nil {
				return eventSchemaErr(e, err)
			}
			if err := requireNumbers(fields, "resolved_at"); err != nil {
				return eventSchemaErr(e, err)
			}
		}
	}
	return nil
}

func objectPayload(e WireEvent) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if len(e.Payload) == 0 {
		return nil, fmt.Errorf("event %q payload must be an object", e.EventID)
	}
	if err := json.Unmarshal(e.Payload, &fields); err != nil {
		return nil, fmt.Errorf("event %q payload must be an object", e.EventID)
	}
	if fields == nil {
		return nil, fmt.Errorf("event %q payload must be an object", e.EventID)
	}
	return fields, nil
}

func requireStrings(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		var s string
		if raw, ok := fields[name]; !ok {
			return fmt.Errorf("missing %s", name)
		} else if err := json.Unmarshal(raw, &s); err != nil || s == "" {
			return fmt.Errorf("%s must be a non-empty string", name)
		}
	}
	return nil
}

func requireNumbers(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		var n float64
		if raw, ok := fields[name]; !ok {
			return fmt.Errorf("missing %s", name)
		} else if err := json.Unmarshal(raw, &n); err != nil {
			return fmt.Errorf("%s must be a number", name)
		}
	}
	return nil
}

func eventSchemaErr(e WireEvent, err error) error {
	return fmt.Errorf("event %q %s payload invalid: %w", e.EventID, e.Type, err)
}

func (s *Server) validateEventCompatibility(source *Node, events []WireEvent) error {
	for _, e := range events {
		switch e.Type {
		case "debt.created":
			var p struct {
				CreditorRef string `json:"creditor_ref"`
				DebtorRef   string `json:"debtor_ref"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			if err := s.validateRefsSameGame(source, p.CreditorRef, p.DebtorRef); err != nil {
				return err
			}
		case "pvp.resolved":
			var p struct {
				AttackerGlobalID string `json:"attacker_global_id"`
				VictimGlobalID   string `json:"victim_global_id"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			if err := s.validateRefsSameGame(source, p.AttackerGlobalID, p.VictimGlobalID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) validateRefsSameGame(source *Node, refs ...string) error {
	for _, ref := range refs {
		nodeID := refNode(ref)
		if nodeID == "" || nodeID == "npc" {
			continue
		}
		n, err := s.store.NodeByID(nodeID)
		if err != nil {
			return err
		}
		if n.GameID != source.GameID {
			return ErrIncompatibleGame
		}
	}
	return nil
}

func (s *Server) handleEventsPull(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	after := queryInt(r, "after", 0)
	limit := queryInt(r, "limit", 500)
	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}
	exclude := ""
	if r.URL.Query().Get("exclude_self") == "true" {
		exclude = n.NodeID
	}
	events, head, err := s.store.FeedSince(after, int(limit), exclude)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "feed failed")
		return
	}
	if events == nil {
		events = []FeedEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"head": head, "events": events})
}

// ---- cross-node PvP (§8) ----

type pvpQueueReq struct {
	AttackerID string          `json:"attacker_id"`
	VictimID   string          `json:"victim_id"`
	Attacker   json.RawMessage `json:"attacker"`
}

func victimNode(victimID string) string {
	return refNode(victimID)
}

func refNode(id string) string {
	if i := strings.Index(id, ":"); i > 0 {
		return id[:i]
	}
	return ""
}

func (s *Server) handlePvPQueue(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	var req pvpQueueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.AttackerID == "" || req.VictimID == "" {
		writeErr(w, http.StatusBadRequest, "attacker_id and victim_id required")
		return
	}
	if !rawObject(req.Attacker) {
		writeErr(w, http.StatusBadRequest, "attacker payload must be an object")
		return
	}
	if !strings.HasPrefix(req.AttackerID, n.NodeID+":") {
		writeErr(w, http.StatusForbidden, "attacker must belong to authenticated node")
		return
	}
	vNode := victimNode(req.VictimID)
	if vNode == "" {
		writeErr(w, http.StatusBadRequest, "invalid victim_id")
		return
	}
	victim, err := s.store.NodeByID(vNode)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "victim node not registered")
			return
		}
		writeErr(w, http.StatusInternalServerError, "victim lookup failed")
		return
	}
	if victim.GameID != n.GameID {
		writeErr(w, http.StatusConflict, "victim node game_id is incompatible")
		return
	}
	if victim.Status != "active" {
		writeErr(w, http.StatusConflict, "victim node is not active")
		return
	}
	// Generate request ID
	var b [8]byte
	_, _ = rand.Read(b[:])
	reqID := hex.EncodeToString(b[:])
	pvpReq := PvPRequest{
		RequestID: reqID, AttackerID: req.AttackerID, VictimID: req.VictimID,
		VictimNode: vNode, AttackerPayload: req.Attacker, CreatedAt: time.Now().Unix(),
	}
	if err := s.store.QueuePvP(pvpReq); err != nil {
		writeErr(w, http.StatusInternalServerError, "queue failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"request_id": reqID, "status": "queued"})
}

func (s *Server) handlePvPPending(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	reqs, err := s.store.PendingPvP(n.NodeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	if reqs == nil {
		reqs = []PvPRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": reqs})
}

func (s *Server) handlePvPResult(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	reqID := r.PathValue("id")
	if reqID == "" {
		writeErr(w, http.StatusBadRequest, "request id required")
		return
	}
	if err := s.store.CompletePvP(reqID, n.NodeID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "pending pvp request not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "complete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

type blockReq struct {
	Error string `json:"error"`
}

func (s *Server) handlePvPBlocked(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	reqID := r.PathValue("id")
	if reqID == "" {
		writeErr(w, http.StatusBadRequest, "request id required")
		return
	}
	if err := s.store.BlockPvP(reqID, n.NodeID, readBlockReason(r)); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "pending pvp request not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "block failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

// ---- cross-node travel (§9) ----

type travelSubmitReq struct {
	GlobalID string          `json:"global_id"`
	HomeNode string          `json:"home_node"`
	DestNode string          `json:"dest_node"`
	Snapshot json.RawMessage `json:"snapshot"`
}

func (s *Server) handleTravelSubmit(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	var req travelSubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.GlobalID == "" || req.DestNode == "" || req.Snapshot == nil {
		writeErr(w, http.StatusBadRequest, "global_id, dest_node, and snapshot required")
		return
	}
	if !rawObject(req.Snapshot) {
		writeErr(w, http.StatusBadRequest, "snapshot must be an object")
		return
	}
	dest, err := s.store.NodeByID(req.DestNode)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "destination node not registered")
			return
		}
		writeErr(w, http.StatusInternalServerError, "destination lookup failed")
		return
	}
	if dest.GameID != n.GameID {
		writeErr(w, http.StatusConflict, "destination node game_id is incompatible")
		return
	}
	if dest.Status != "active" {
		writeErr(w, http.StatusConflict, "destination node is not active")
		return
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	travelID := hex.EncodeToString(b[:])
	tr := TravelRequest{
		TravelID: travelID, GlobalID: req.GlobalID, HomeNode: req.HomeNode,
		FromNode: n.NodeID, DestNode: req.DestNode,
		Snapshot: req.Snapshot, CreatedAt: time.Now().Unix(),
	}
	if err := s.store.SubmitTravel(tr); err != nil {
		if errors.Is(err, ErrTravelActive) {
			writeErr(w, http.StatusConflict, "player already in transit")
			return
		}
		if errors.Is(err, ErrLocationMismatch) {
			writeErr(w, http.StatusForbidden, "player is not located at this node")
			return
		}
		writeErr(w, http.StatusInternalServerError, "submit failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"travel_id": travelID, "status": "pending"})
}

func (s *Server) handleTravelPending(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	reqs, err := s.store.PendingTravel(n.NodeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	if reqs == nil {
		reqs = []TravelRequest{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"arrivals": reqs})
}

func (s *Server) handleTravelArrived(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	travelID := r.PathValue("id")
	if travelID == "" {
		writeErr(w, http.StatusBadRequest, "travel id required")
		return
	}
	if err := s.store.ArriveTravel(travelID, n.NodeID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "pending travel request not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "arrive failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "arrived"})
}

func (s *Server) handleTravelBlocked(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	travelID := r.PathValue("id")
	if travelID == "" {
		writeErr(w, http.StatusBadRequest, "travel id required")
		return
	}
	if err := s.store.BlockTravel(travelID, n.NodeID, readBlockReason(r)); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "pending travel request not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "block failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

func readBlockReason(r *http.Request) string {
	var req blockReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := strings.TrimSpace(req.Error)
	if reason == "" {
		return "blocked by destination node"
	}
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return reason
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

// ---- cross-node Ledger (§7) ----

func (s *Server) handleDebts(w http.ResponseWriter, r *http.Request) {
	debtor := r.URL.Query().Get("debtor")
	if debtor == "" {
		writeErr(w, http.StatusBadRequest, "debtor query param required")
		return
	}
	debts, err := s.store.DebtsForDebtor(debtor)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ledger query failed")
		return
	}
	if debts == nil {
		debts = []HubDebt{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"debtor": debtor, "debts": debts})
}

// ---- roster (§6) ----

type rosterPushReq struct {
	Entries []RosterEntry `json:"entries"`
}

func (s *Server) handleRosterPush(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	var req rosterPushReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	// Enforce source integrity: every entry must belong to the authenticated node.
	for i := range req.Entries {
		req.Entries[i].NodeID = n.NodeID
	}
	if err := s.store.UpsertRoster(n.NodeID, req.Entries); err != nil {
		writeErr(w, http.StatusInternalServerError, "roster update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": len(req.Entries)})
}

func (s *Server) handleRosterPull(w http.ResponseWriter, r *http.Request) {
	n := nodeFrom(r.Context())
	exclude := ""
	if r.URL.Query().Get("exclude_self") == "true" {
		exclude = n.NodeID
	}
	entries, err := s.store.GetRoster(exclude)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "roster fetch failed")
		return
	}
	if entries == nil {
		entries = []RosterEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func queryInt(r *http.Request, key string, def int64) int64 {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

// ---- root index (no auth) ----

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hub":              "InterDOOR federation hub",
		"protocol_version": ProtocolVersion,
		"endpoints": map[string]string{
			"status":    "/v1/status",
			"directory": "/v1/directory",
		},
	})
}

// ---- public directory + status (no auth) ----

// onlineThreshold is how recently a node must have heartbeated to be considered online.
const onlineThreshold = 5 * time.Minute

type directoryEntry struct {
	NodeID        string `json:"node_id"`
	GameID        string `json:"game_id"`
	GameTitle     string `json:"game_title"`
	GameVersion   string `json:"game_version"`
	AdvertiseAddr string `json:"advertise_addr"`
	PlayerCount   int    `json:"player_count"`
	LastHeartbeat int64  `json:"last_heartbeat"`
	Online        bool   `json:"online"`
}

func (s *Server) handleDirectory(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.GetDirectory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "directory unavailable")
		return
	}
	now := time.Now()
	entries := make([]directoryEntry, len(nodes))
	online := 0
	for i, n := range nodes {
		isOnline := now.Sub(n.LastHeartbeat) <= onlineThreshold
		if isOnline {
			online++
		}
		entries[i] = directoryEntry{
			NodeID:        n.NodeID,
			GameID:        n.GameID,
			GameTitle:     n.DisplayGameTitle(),
			GameVersion:   n.GameVersion,
			AdvertiseAddr: n.AdvertiseAddr,
			PlayerCount:   n.PlayerCount,
			LastHeartbeat: n.LastHeartbeat.Unix(),
			Online:        isOnline,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": entries, "total": len(entries), "online": online,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.GetDirectory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "status unavailable")
		return
	}
	eventCount, _ := s.store.EventCount()
	now := time.Now()
	online := 0
	for _, n := range nodes {
		if now.Sub(n.LastHeartbeat) <= onlineThreshold {
			online++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hub":              "ok",
		"protocol_version": ProtocolVersion,
		"nodes_total":      len(nodes),
		"nodes_online":     online,
		"events_total":     eventCount,
	})
}

// ---- auth middleware ----

type ctxKey int

const nodeCtxKey ctxKey = 0

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := bearer(r)
		if tok == "" {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		n, err := s.store.NodeByAPIKeyHash(hashKey(tok))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		if n.Status != "active" {
			writeErr(w, http.StatusForbidden, "node is not active")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), nodeCtxKey, n)))
	}
}

func nodeFrom(ctx context.Context) *Node {
	n, _ := ctx.Value(nodeCtxKey).(*Node)
	return n
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
