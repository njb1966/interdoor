package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestHub(t *testing.T) *httptest.Server {
	t.Helper()
	ts, _ := newTestHubWithStore(t)
	return ts
}

func newTestHubWithStore(t *testing.T) (*httptest.Server, *sqliteStore) {
	t.Helper()
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqliteStore := store.(*sqliteStore)
	ts := httptest.NewServer(NewServer(store, "regsecret").Handler())
	t.Cleanup(func() { ts.Close(); _ = store.Close() })
	return ts, sqliteStore
}

func post(t *testing.T, url, bearer, body string) (int, []byte) {
	t.Helper()
	return do(t, http.MethodPost, url, bearer, strings.NewReader(body))
}

func get(t *testing.T, url, bearer string) (int, []byte) {
	t.Helper()
	return do(t, http.MethodGet, url, bearer, nil)
}

func registerNode(t *testing.T, baseURL, nodeID, gameID string) string {
	t.Helper()
	body := `{"node_id":"` + nodeID + `","registration_token":"regsecret","game_id":"` + gameID + `","game_version":"1.0.0","protocol_version":"1"}`
	code, resp := post(t, baseURL+"/v1/register", "", body)
	if code != http.StatusOK {
		t.Fatalf("register %s: want 200, got %d (%s)", nodeID, code, resp)
	}
	var rr registerResp
	if err := json.Unmarshal(resp, &rr); err != nil || rr.APIKey == "" {
		t.Fatalf("register %s response: %v (%s)", nodeID, err, resp)
	}
	return rr.APIKey
}

func pendingPvPCount(t *testing.T, store *sqliteStore, nodeID string) int {
	t.Helper()
	n, err := store.PendingPvPCount(nodeID)
	if err != nil {
		t.Fatalf("pending pvp count: %v", err)
	}
	return n
}

func pendingTravelCount(t *testing.T, store *sqliteStore, nodeID string) int {
	t.Helper()
	n, err := store.PendingTravelCount(nodeID)
	if err != nil {
		t.Fatalf("pending travel count: %v", err)
	}
	return n
}

func do(t *testing.T, method, url, bearer string, body io.Reader) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func TestPvPQueueValidationAndCompletionOwnership(t *testing.T) {
	ts := newTestHub(t)
	attackerKey := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	victimKey := registerNode(t, ts.URL, "node02", "ledger_of_the_low")
	otherGameKey := registerNode(t, ts.URL, "node03", "empire_ascendant")

	body := `{"attacker_id":"node01:p_a","victim_id":"node02:p_v","attacker":{}}`
	code, resp := post(t, ts.URL+"/v1/pvp", attackerKey, body)
	if code != http.StatusOK {
		t.Fatalf("queue pvp: want 200, got %d (%s)", code, resp)
	}
	var queued struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(resp, &queued)
	if queued.RequestID == "" {
		t.Fatalf("queue response missing request_id: %s", resp)
	}

	if code, resp := post(t, ts.URL+"/v1/pvp/"+queued.RequestID+"/result", attackerKey, `{}`); code != http.StatusNotFound {
		t.Fatalf("attacker completion: want 404, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp/"+queued.RequestID+"/result", victimKey, `{}`); code != http.StatusOK {
		t.Fatalf("victim completion: want 200, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp/"+queued.RequestID+"/result", victimKey, `{}`); code != http.StatusNotFound {
		t.Fatalf("duplicate completion: want 404, got %d (%s)", code, resp)
	}

	crossGame := `{"attacker_id":"node01:p_a","victim_id":"node03:p_v","attacker":{}}`
	if code, resp := post(t, ts.URL+"/v1/pvp", attackerKey, crossGame); code != http.StatusConflict {
		t.Fatalf("cross-game pvp: want 409, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp", otherGameKey, body); code != http.StatusForbidden {
		t.Fatalf("spoofed attacker: want 403, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp", attackerKey, `{"attacker_id":"node01:p_a","victim_id":"node02:p_v","attacker":null}`); code != http.StatusBadRequest {
		t.Fatalf("non-object attacker payload: want 400, got %d (%s)", code, resp)
	}
}

func TestQueueBlockedLifecycle(t *testing.T) {
	ts, store := newTestHubWithStore(t)
	attackerKey := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	victimKey := registerNode(t, ts.URL, "node02", "ledger_of_the_low")

	pvpBody := `{"attacker_id":"node01:p_a","victim_id":"node02:p_v","attacker":{}}`
	code, resp := post(t, ts.URL+"/v1/pvp", attackerKey, pvpBody)
	if code != http.StatusOK {
		t.Fatalf("queue pvp: want 200, got %d (%s)", code, resp)
	}
	var pvpQueued struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(resp, &pvpQueued)
	if code, resp := post(t, ts.URL+"/v1/pvp/"+pvpQueued.RequestID+"/blocked", attackerKey, `{"error":"wrong owner"}`); code != http.StatusNotFound {
		t.Fatalf("attacker block pvp: want 404, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp/"+pvpQueued.RequestID+"/blocked", victimKey, `{"error":"bad attacker payload"}`); code != http.StatusOK {
		t.Fatalf("victim block pvp: want 200, got %d (%s)", code, resp)
	}
	if got := pendingPvPCount(t, store, "node02"); got != 0 {
		t.Fatalf("blocked pvp should leave pending queue, got %d", got)
	}
	var pvpStatus, pvpErr string
	if err := store.db.QueryRow(`SELECT status,error FROM pvp_requests WHERE request_id=?`, pvpQueued.RequestID).Scan(&pvpStatus, &pvpErr); err != nil {
		t.Fatalf("read blocked pvp: %v", err)
	}
	if pvpStatus != "blocked" || pvpErr != "bad attacker payload" {
		t.Fatalf("blocked pvp row: status=%q error=%q", pvpStatus, pvpErr)
	}

	travelBody := `{"global_id":"node01:p_x","home_node":"node01","dest_node":"node02","snapshot":{}}`
	code, resp = post(t, ts.URL+"/v1/travel", attackerKey, travelBody)
	if code != http.StatusOK {
		t.Fatalf("queue travel: want 200, got %d (%s)", code, resp)
	}
	var travelQueued struct {
		TravelID string `json:"travel_id"`
	}
	_ = json.Unmarshal(resp, &travelQueued)
	if code, resp := post(t, ts.URL+"/v1/travel/"+travelQueued.TravelID+"/blocked", attackerKey, `{"error":"wrong owner"}`); code != http.StatusNotFound {
		t.Fatalf("origin block travel: want 404, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/travel/"+travelQueued.TravelID+"/blocked", victimKey, `{"error":"bad snapshot"}`); code != http.StatusOK {
		t.Fatalf("dest block travel: want 200, got %d (%s)", code, resp)
	}
	if got := pendingTravelCount(t, store, "node02"); got != 0 {
		t.Fatalf("blocked travel should leave pending queue, got %d", got)
	}
	var travelStatus, travelErr string
	if err := store.db.QueryRow(`SELECT status,error FROM travel WHERE travel_id=?`, travelQueued.TravelID).Scan(&travelStatus, &travelErr); err != nil {
		t.Fatalf("read blocked travel: %v", err)
	}
	if travelStatus != "blocked" || travelErr != "bad snapshot" {
		t.Fatalf("blocked travel row: status=%q error=%q", travelStatus, travelErr)
	}

	if code, resp := post(t, ts.URL+"/v1/travel", attackerKey, `{"global_id":"node01:p_y","home_node":"node01","dest_node":"node02","snapshot":null}`); code != http.StatusBadRequest {
		t.Fatalf("non-object travel snapshot: want 400, got %d (%s)", code, resp)
	}
}

func TestDebtEventRejectsCrossGamePlayerRefs(t *testing.T) {
	ts := newTestHub(t)
	node1Key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	registerNode(t, ts.URL, "node02", "ledger_of_the_low")
	registerNode(t, ts.URL, "node03", "empire_ascendant")

	okDebt := `{"events":[{"event_id":"node01:1","source_node":"node01","seq":1,"type":"debt.created","ts":1,"payload":{"obligation_id":"node01:o_1","creditor_ref":"npc:npc_maren","debtor_ref":"node02:p_b","kind":"debt","terms":"same game","weight":5}}]}`
	if code, resp := post(t, ts.URL+"/v1/events", node1Key, okDebt); code != http.StatusOK {
		t.Fatalf("same-game debt event: want 200, got %d (%s)", code, resp)
	}

	badDebt := `{"events":[{"event_id":"node01:2","source_node":"node01","seq":2,"type":"debt.created","ts":2,"payload":{"obligation_id":"node01:o_2","creditor_ref":"npc:npc_maren","debtor_ref":"node03:p_b","kind":"debt","terms":"cross game","weight":5}}]}`
	if code, resp := post(t, ts.URL+"/v1/events", node1Key, badDebt); code != http.StatusConflict {
		t.Fatalf("cross-game debt event: want 409, got %d (%s)", code, resp)
	}
}

func TestMalformedStandardEventPayloadIsRejected(t *testing.T) {
	ts := newTestHub(t)
	node1Key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	registerNode(t, ts.URL, "node02", "ledger_of_the_low")

	push := `{"events":[{"event_id":"node01:1","source_node":"node01","seq":1,"type":"debt.created","ts":1,"payload":"not-an-object"}]}`
	code, body := post(t, ts.URL+"/v1/events", node1Key, push)
	if code != http.StatusBadRequest {
		t.Fatalf("malformed debt push: want 400, got %d (%s)", code, body)
	}
}

func TestRawHTTPClientConformanceTier1Tier2(t *testing.T) {
	ts := newTestHub(t)
	node1Key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	node2Key := registerNode(t, ts.URL, "node02", "ledger_of_the_low")

	if code, resp := get(t, ts.URL+"/v1/status", ""); code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (%s)", code, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/directory", ""); code != http.StatusOK {
		t.Fatalf("directory: want 200, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/heartbeat", node1Key, `{"node_id":"node01","player_count":1,"uptime_s":10,"game_version":"1.0.0"}`); code != http.StatusOK {
		t.Fatalf("heartbeat: want 200, got %d (%s)", code, resp)
	}

	roster := `{"entries":[{"global_id":"node01:p_a","name":"Ada","level":2,"status":"active","last_seen":1}]}`
	if code, resp := post(t, ts.URL+"/v1/roster", node1Key, roster); code != http.StatusOK {
		t.Fatalf("roster push: want 200, got %d (%s)", code, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/roster?exclude_self=true", node2Key); code != http.StatusOK || !strings.Contains(string(resp), "node01:p_a") {
		t.Fatalf("roster pull: want node01 player, got %d (%s)", code, resp)
	}

	eventPush := `{"events":[{"event_id":"node01:1","source_node":"node01","seq":1,"type":"debt.created","ts":1,"payload":{"obligation_id":"node01:o_1","creditor_ref":"npc:npc_maren","debtor_ref":"node02:p_b","kind":"debt","terms":"raw http","weight":5}}]}`
	if code, resp := post(t, ts.URL+"/v1/events", node1Key, eventPush); code != http.StatusOK {
		t.Fatalf("event push: want 200, got %d (%s)", code, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/events?after=0&exclude_self=true", node2Key); code != http.StatusOK || !strings.Contains(string(resp), "debt.created") {
		t.Fatalf("event pull: want debt.created, got %d (%s)", code, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/debts?debtor=node02:p_b", node2Key); code != http.StatusOK || !strings.Contains(string(resp), "node01:o_1") {
		t.Fatalf("debt query: want obligation, got %d (%s)", code, resp)
	}

	pvp := `{"attacker_id":"node01:p_a","victim_id":"node02:p_b","attacker":{"level":2}}`
	code, resp := post(t, ts.URL+"/v1/pvp", node1Key, pvp)
	if code != http.StatusOK {
		t.Fatalf("pvp queue: want 200, got %d (%s)", code, resp)
	}
	var pvpQueued struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(resp, &pvpQueued); err != nil || pvpQueued.RequestID == "" {
		t.Fatalf("pvp queue response: %v (%s)", err, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/pvp/pending", node2Key); code != http.StatusOK || !strings.Contains(string(resp), pvpQueued.RequestID) {
		t.Fatalf("pvp pending: want request, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/pvp/"+pvpQueued.RequestID+"/result", node2Key, `{}`); code != http.StatusOK {
		t.Fatalf("pvp result: want 200, got %d (%s)", code, resp)
	}

	travel := `{"global_id":"node01:p_a","home_node":"node01","dest_node":"node02","snapshot":{"global_id":"node01:p_a"}}`
	code, resp = post(t, ts.URL+"/v1/travel", node1Key, travel)
	if code != http.StatusOK {
		t.Fatalf("travel submit: want 200, got %d (%s)", code, resp)
	}
	var travelQueued struct {
		TravelID string `json:"travel_id"`
	}
	if err := json.Unmarshal(resp, &travelQueued); err != nil || travelQueued.TravelID == "" {
		t.Fatalf("travel response: %v (%s)", err, resp)
	}
	if code, resp := get(t, ts.URL+"/v1/travel/pending", node2Key); code != http.StatusOK || !strings.Contains(string(resp), travelQueued.TravelID) {
		t.Fatalf("travel pending: want arrival, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/travel/"+travelQueued.TravelID+"/arrived", node2Key, `{}`); code != http.StatusOK {
		t.Fatalf("travel arrived: want 200, got %d (%s)", code, resp)
	}
}

func TestStandardEventSchemaValidation(t *testing.T) {
	ts := newTestHub(t)
	node1Key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	registerNode(t, ts.URL, "node02", "ledger_of_the_low")

	valid := []struct {
		typ     string
		payload string
	}{
		{"player.created", `{"global_id":"node01:p_1","name":"Ada","home_node":"node01","created_at":1}`},
		{"player.died", `{"global_id":"node01:p_1","cause":{"type":"test"},"timestamp":2}`},
		{"player.traveled", `{"global_id":"node01:p_1","src_node":"node01","dest_node":"node02","snapshot_hash":"abc","timestamp":3}`},
		{"debt.created", `{"obligation_id":"node01:o_1","creditor_ref":"npc:npc_maren","debtor_ref":"node02:p_b","kind":"debt","terms":"same game","weight":5}`},
		{"debt.adjusted", `{"obligation_id":"node01:o_1","old_weight":5,"new_weight":3,"delta":-2,"reason":"partial_payment"}`},
		{"debt.resolved", `{"obligation_id":"node01:o_1","resolution":"paid","resolved_at":4}`},
		{"pvp.resolved", `{"request_id":"req1","attacker_global_id":"node01:p_a","victim_global_id":"node02:p_v","winner_global_id":"node01:p_a","result_text":"win","resolved_at":5}`},
	}
	for i, tc := range valid {
		seq := i + 1
		body := fmt.Sprintf(`{"events":[{"event_id":"node01:%d","source_node":"node01","seq":%d,"type":%q,"ts":%d,"payload":%s}]}`,
			seq, seq, tc.typ, seq, tc.payload)
		if code, resp := post(t, ts.URL+"/v1/events", node1Key, body); code != http.StatusOK {
			t.Fatalf("%s valid event: want 200, got %d (%s)", tc.typ, code, resp)
		}
	}

	invalid := `{"events":[{"event_id":"node01:bad","source_node":"node01","seq":99,"type":"player.created","ts":99,"payload":{"global_id":"node01:p_bad","name":"Bad"}}]}`
	if code, resp := post(t, ts.URL+"/v1/events", node1Key, invalid); code != http.StatusBadRequest {
		t.Fatalf("invalid standard event: want 400, got %d (%s)", code, resp)
	}
}

func TestTravelCurrentLocationAndCompletionOwnership(t *testing.T) {
	ts := newTestHub(t)
	node1Key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	node2Key := registerNode(t, ts.URL, "node02", "ledger_of_the_low")
	registerNode(t, ts.URL, "node03", "empire_ascendant")

	travel := `{"global_id":"node01:p_x","home_node":"node01","dest_node":"node02","snapshot":{}}`
	code, resp := post(t, ts.URL+"/v1/travel", node1Key, travel)
	if code != http.StatusOK {
		t.Fatalf("submit travel: want 200, got %d (%s)", code, resp)
	}
	var queued struct {
		TravelID string `json:"travel_id"`
	}
	_ = json.Unmarshal(resp, &queued)
	if queued.TravelID == "" {
		t.Fatalf("travel response missing travel_id: %s", resp)
	}

	if code, resp := post(t, ts.URL+"/v1/travel/"+queued.TravelID+"/arrived", node1Key, `{}`); code != http.StatusNotFound {
		t.Fatalf("origin arrival: want 404, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/travel/"+queued.TravelID+"/arrived", node2Key, `{}`); code != http.StatusOK {
		t.Fatalf("dest arrival: want 200, got %d (%s)", code, resp)
	}

	returnHome := `{"global_id":"node01:p_x","home_node":"node01","dest_node":"node01","snapshot":{}}`
	if code, resp := post(t, ts.URL+"/v1/travel", node1Key, returnHome); code != http.StatusForbidden {
		t.Fatalf("non-current origin travel: want 403, got %d (%s)", code, resp)
	}
	if code, resp := post(t, ts.URL+"/v1/travel", node2Key, returnHome); code != http.StatusOK {
		t.Fatalf("current holder return travel: want 200, got %d (%s)", code, resp)
	}

	crossGame := `{"global_id":"node01:p_y","home_node":"node01","dest_node":"node03","snapshot":{}}`
	if code, resp := post(t, ts.URL+"/v1/travel", node1Key, crossGame); code != http.StatusConflict {
		t.Fatalf("cross-game travel: want 409, got %d (%s)", code, resp)
	}
}

func TestEventFeed(t *testing.T) {
	ts := newTestHub(t)
	_, body := post(t, ts.URL+"/v1/register", "",
		`{"node_id":"node01","registration_token":"regsecret","game_id":"g","protocol_version":"1"}`)
	var rr registerResp
	_ = json.Unmarshal(body, &rr)

	push := `{"events":[
		{"event_id":"node01:1","source_node":"node01","seq":1,"type":"player.created","ts":1,"payload":{"global_id":"node01:p_1","name":"Ada","home_node":"node01","created_at":1}},
		{"event_id":"node01:2","source_node":"node01","seq":2,"type":"player.died","ts":2,"payload":{"global_id":"node01:p_1","cause":{"type":"test"},"timestamp":2}}]}`

	// Push two events.
	code, body := post(t, ts.URL+"/v1/events", rr.APIKey, push)
	if code != http.StatusOK {
		t.Fatalf("push: %d (%s)", code, body)
	}
	var pr struct {
		Accepted   int   `json:"accepted"`
		LastHubSeq int64 `json:"last_hub_seq"`
	}
	_ = json.Unmarshal(body, &pr)
	if pr.Accepted != 2 {
		t.Fatalf("accepted: want 2, got %d", pr.Accepted)
	}

	// Re-push is deduped.
	_, body = post(t, ts.URL+"/v1/events", rr.APIKey, push)
	_ = json.Unmarshal(body, &pr)
	if pr.Accepted != 0 {
		t.Fatalf("dedup: want 0 accepted, got %d", pr.Accepted)
	}

	// A node cannot push events from another source.
	bad := `{"events":[{"event_id":"node02:1","source_node":"node02","seq":1,"type":"x","ts":1,"payload":{}}]}`
	if code, _ := post(t, ts.URL+"/v1/events", rr.APIKey, bad); code != http.StatusForbidden {
		t.Fatalf("source mismatch: want 403, got %d", code)
	}

	// Pull returns both events.
	var fr struct {
		Head   int64            `json:"head"`
		Events []map[string]any `json:"events"`
	}
	_, body = get(t, ts.URL+"/v1/events?after=0", rr.APIKey)
	_ = json.Unmarshal(body, &fr)
	if len(fr.Events) != 2 {
		t.Fatalf("pull: want 2 events, got %d", len(fr.Events))
	}

	// exclude_self omits the caller's own events.
	_, body = get(t, ts.URL+"/v1/events?after=0&exclude_self=true", rr.APIKey)
	_ = json.Unmarshal(body, &fr)
	if len(fr.Events) != 0 {
		t.Fatalf("exclude_self: want 0, got %d", len(fr.Events))
	}
}

func TestDirectoryAndStatus(t *testing.T) {
	ts := newTestHub(t)

	// Directory and status are public (no auth).
	code, body := get(t, ts.URL+"/v1/directory", "")
	if code != http.StatusOK {
		t.Fatalf("directory: want 200, got %d (%s)", code, body)
	}
	var dir struct {
		Nodes  []map[string]any `json:"nodes"`
		Total  int              `json:"total"`
		Online int              `json:"online"`
	}
	_ = json.Unmarshal(body, &dir)
	if dir.Total != 0 {
		t.Fatalf("empty directory: want total=0, got %d", dir.Total)
	}

	// Register a node so it appears in the directory.
	_, body = post(t, ts.URL+"/v1/register", "",
		`{"node_id":"node01","registration_token":"regsecret","game_id":"ledger_of_the_low","game_title":"Ledger of the Low","game_version":"1.0.0","protocol_version":"1","advertise_addr":"ssh://node01.example.com:2323"}`)
	var rr registerResp
	_ = json.Unmarshal(body, &rr)

	// Heartbeat so the node is considered online.
	post(t, ts.URL+"/v1/heartbeat", rr.APIKey, `{"node_id":"node01","player_count":5,"uptime_s":60,"game_version":"1.0.0"}`)

	code, body = get(t, ts.URL+"/v1/directory", "")
	if code != http.StatusOK {
		t.Fatalf("directory after register: %d", code)
	}
	_ = json.Unmarshal(body, &dir)
	if dir.Total != 1 {
		t.Fatalf("want total=1, got %d", dir.Total)
	}
	if dir.Online != 1 {
		t.Fatalf("want online=1, got %d", dir.Online)
	}
	if addr, _ := dir.Nodes[0]["advertise_addr"].(string); addr != "ssh://node01.example.com:2323" {
		t.Fatalf("advertise_addr: got %q", addr)
	}
	if gameID, _ := dir.Nodes[0]["game_id"].(string); gameID != "ledger_of_the_low" {
		t.Fatalf("game_id: got %q", gameID)
	}
	if gameTitle, _ := dir.Nodes[0]["game_title"].(string); gameTitle != "Ledger of the Low" {
		t.Fatalf("game_title: got %q", gameTitle)
	}

	// Status endpoint.
	code, body = get(t, ts.URL+"/v1/status", "")
	if code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", code)
	}
	var status map[string]any
	_ = json.Unmarshal(body, &status)
	if status["hub"] != "ok" {
		t.Fatalf("status hub: want ok, got %v", status["hub"])
	}
	if int(status["nodes_total"].(float64)) != 1 {
		t.Fatalf("status nodes_total: want 1, got %v", status["nodes_total"])
	}
}

func TestSuspendedNodeIsHiddenAndCannotAuthenticate(t *testing.T) {
	ts, store := newTestHubWithStore(t)
	key := registerNode(t, ts.URL, "node01", "ledger_of_the_low")
	if code, resp := post(t, ts.URL+"/v1/heartbeat", key, `{"node_id":"node01","player_count":1,"uptime_s":1,"game_version":"1.0.0"}`); code != http.StatusOK {
		t.Fatalf("heartbeat before suspend: want 200, got %d (%s)", code, resp)
	}

	if _, err := store.db.Exec(`UPDATE nodes SET status='suspended' WHERE node_id='node01'`); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	if code, resp := post(t, ts.URL+"/v1/heartbeat", key, `{"node_id":"node01","player_count":1,"uptime_s":2,"game_version":"1.0.0"}`); code != http.StatusForbidden {
		t.Fatalf("heartbeat after suspend: want 403, got %d (%s)", code, resp)
	}

	code, body := get(t, ts.URL+"/v1/directory", "")
	if code != http.StatusOK {
		t.Fatalf("directory: want 200, got %d (%s)", code, body)
	}
	var dir struct {
		Total int `json:"total"`
	}
	_ = json.Unmarshal(body, &dir)
	if dir.Total != 0 {
		t.Fatalf("suspended node should be hidden from directory, got total=%d", dir.Total)
	}

	code, body = get(t, ts.URL+"/v1/status", "")
	if code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (%s)", code, body)
	}
	var status map[string]any
	_ = json.Unmarshal(body, &status)
	if int(status["nodes_total"].(float64)) != 0 {
		t.Fatalf("suspended node should be excluded from status, got nodes_total=%v", status["nodes_total"])
	}
}

func TestTravelConflict(t *testing.T) {
	ts := newTestHub(t)
	_, body := post(t, ts.URL+"/v1/register", "",
		`{"node_id":"node01","registration_token":"regsecret","game_id":"g","protocol_version":"1"}`)
	var rr registerResp
	_ = json.Unmarshal(body, &rr)
	registerNode(t, ts.URL, "node02", "g")

	travel := `{"global_id":"node01:p_x","home_node":"node01","dest_node":"node02","snapshot":{}}`

	// First submission succeeds.
	code, body := post(t, ts.URL+"/v1/travel", rr.APIKey, travel)
	if code != http.StatusOK {
		t.Fatalf("first submit: want 200, got %d (%s)", code, body)
	}

	// Second submission for the same player is rejected (single-active invariant §10).
	code, body = post(t, ts.URL+"/v1/travel", rr.APIKey, travel)
	if code != http.StatusConflict {
		t.Fatalf("double submit: want 409, got %d (%s)", code, body)
	}
}

func TestHeartbeatPendingCounts(t *testing.T) {
	ts := newTestHub(t)
	_, body := post(t, ts.URL+"/v1/register", "",
		`{"node_id":"node01","registration_token":"regsecret","game_id":"g","protocol_version":"1"}`)
	var rr registerResp
	_ = json.Unmarshal(body, &rr)

	hb := `{"node_id":"node01","player_count":0,"uptime_s":0,"game_version":"1.0.0"}`

	// Initially hub_seq_head=0, no pending work.
	_, body = post(t, ts.URL+"/v1/heartbeat", rr.APIKey, hb)
	var resp heartbeatResp
	_ = json.Unmarshal(body, &resp)
	if resp.HubSeqHead != 0 {
		t.Fatalf("empty hub: want head=0, got %d", resp.HubSeqHead)
	}
	if resp.Pending.PVP != 0 || resp.Pending.Travel != 0 {
		t.Fatalf("empty: want 0 pending, got pvp=%d travel=%d", resp.Pending.PVP, resp.Pending.Travel)
	}

	// Push an event; hub_seq_head should advance.
	push := `{"events":[{"event_id":"node01:1","source_node":"node01","seq":1,"type":"x","ts":1,"payload":{}}]}`
	post(t, ts.URL+"/v1/events", rr.APIKey, push)

	_, body = post(t, ts.URL+"/v1/heartbeat", rr.APIKey, hb)
	_ = json.Unmarshal(body, &resp)
	if resp.HubSeqHead != 1 {
		t.Fatalf("after push: want head=1, got %d", resp.HubSeqHead)
	}
}

func TestRegisterAndHeartbeat(t *testing.T) {
	ts := newTestHub(t)
	reg := `{"node_id":"node01","registration_token":"regsecret","game_id":"ledger_of_the_low","game_version":"1.0.0","protocol_version":"1"}`

	// Register succeeds and returns an api key.
	code, body := post(t, ts.URL+"/v1/register", "", reg)
	if code != http.StatusOK {
		t.Fatalf("register: want 200, got %d (%s)", code, body)
	}
	var rr registerResp
	if err := json.Unmarshal(body, &rr); err != nil || rr.APIKey == "" {
		t.Fatalf("register response: %v (%s)", err, body)
	}

	// Duplicate node id is rejected.
	if code, _ := post(t, ts.URL+"/v1/register", "", reg); code != http.StatusConflict {
		t.Fatalf("duplicate register: want 409, got %d", code)
	}

	// Bad registration token is rejected.
	bad := `{"node_id":"node02","registration_token":"wrong","game_id":"g","protocol_version":"1"}`
	if code, _ := post(t, ts.URL+"/v1/register", "", bad); code != http.StatusUnauthorized {
		t.Fatalf("bad token: want 401, got %d", code)
	}

	// Wrong protocol version is rejected.
	badproto := `{"node_id":"node03","registration_token":"regsecret","game_id":"g","protocol_version":"99"}`
	if code, _ := post(t, ts.URL+"/v1/register", "", badproto); code != http.StatusUpgradeRequired {
		t.Fatalf("bad protocol: want 426, got %d", code)
	}

	// Heartbeat with the api key succeeds.
	hb := `{"node_id":"node01","player_count":3,"uptime_s":120,"game_version":"1.0.0"}`
	if code, _ := post(t, ts.URL+"/v1/heartbeat", rr.APIKey, hb); code != http.StatusOK {
		t.Fatalf("heartbeat: want 200, got %d", code)
	}

	// Heartbeat without a key is rejected.
	if code, _ := post(t, ts.URL+"/v1/heartbeat", "", hb); code != http.StatusUnauthorized {
		t.Fatalf("unauth heartbeat: want 401, got %d", code)
	}

	// Heartbeat with a bogus key is rejected.
	if code, _ := post(t, ts.URL+"/v1/heartbeat", "not-a-real-key", hb); code != http.StatusUnauthorized {
		t.Fatalf("bad-key heartbeat: want 401, got %d", code)
	}
}
