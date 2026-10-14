package workflow_test

import (
	"bytes"
	"commonreel/internal/workflow"
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type headersTransport struct{ secret, id string }

func (t headersTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("Authorization", "Bearer "+t.secret)
	r.Header.Set("X-CommonReel-Session", t.id)
	return http.DefaultTransport.RoundTrip(r)
}
func TestMCPUsesBoundRoomAndOnlyAllowsSevenTools(t *testing.T) {
	h := newHarness(t, workflow.Config{})
	h.cmd("join_member", map[string]any{"name": "Alex"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "acceptance", Version: "1"}, nil)
	session, e := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: h.server.URL + "/mcp", DisableStandaloneSSE: true, HTTPClient: &http.Client{Transport: headersTransport{secret: h.secret, id: h.id}}}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if e != nil {
		t.Fatal(e)
	}
	defer session.Close()
	tools, e := session.ListTools(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if len(tools.Tools) != 7 {
		t.Fatal("whitelist", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Name == "save_choice" || tool.Name == "remove_member" || tool.Name == "record_acceptance" {
			t.Fatal("forbidden tool", tool.Name)
		}
	}
	res, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "set_member", Arguments: map[string]any{"member_id": "m1", "expected_room_revision": h.rev, "operation_key": "mcp-edit-0001", "patch": map[string]any{"vetoes": []string{"Horror"}}}})
	if e != nil || res.IsError {
		t.Fatal(e, res)
	}
	res, e = session.CallTool(ctx, &mcp.CallToolParams{Name: "set_member", Arguments: map[string]any{"room_id": "some-other-room", "member_id": "m1", "expected_room_revision": h.rev + 1, "operation_key": "mcp-edit-0002", "patch": map[string]any{"vetoes": []string{"Drama"}}}})
	if e == nil && !res.IsError {
		t.Fatal("unknown room field accepted")
	}
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if room["room_revision"] != float64(h.rev+1) {
		t.Fatal("invalid tool changed room", room)
	}
}

func TestAgentUnknownToolCannotSaveAndAmbiguousSpeakerIsClarified(t *testing.T) {
	modelCalls := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelCalls++
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"function": map[string]any{"name": "save_choice", "arguments": map[string]any{}}}}}})
	}))
	defer model.Close()
	h := newHarness(t, workflow.Config{OllamaEndpoint: model.URL, OllamaModel: "controlled", AllowTestModel: true})
	h.app.SetMCPEndpoint(h.server.URL + "/mcp")
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	status, v := h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", map[string]any{"operation_key": "agent-0001", "message": "I do not want horror"})
	if status != 200 || v["status"] != "clarification" || modelCalls != 0 {
		t.Fatal(status, v, modelCalls)
	}
	status, v = h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", map[string]any{"operation_key": "agent-0002", "message": "Set Alex's veto to Horror", "speaker_member_id": "m1"})
	if status != 200 || v["status"] != "tool_rejected" {
		t.Fatal(status, v)
	}
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if room["room_revision"] != float64(h.rev) {
		t.Fatal("agent exceeded authority", room)
	}
}

func TestClosingRoomCancelsPendingLocalModelTurn(t *testing.T) {
	entered := make(chan struct{})
	var once sync.Once
	var capability string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if capability != "" && bytes.Contains(b, []byte(capability)) {
			t.Error("room capability sent to the model")
		}
		once.Do(func() { close(entered) })
		<-r.Context().Done()
	}))
	defer model.Close()
	h := newHarness(t, workflow.Config{OllamaEndpoint: model.URL, OllamaModel: "controlled"})
	capability = h.secret
	h.app.SetMCPEndpoint(h.server.URL + "/mcp")
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	done := make(chan map[string]any, 1)
	go func() {
		_, v := h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", map[string]any{"operation_key": "agent-cancel-001", "message": "Recommend a film for the current room"})
		done <- v
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("model request never started")
	}
	if status, r := h.cmd("close_session", nil); status != 200 {
		t.Fatal(status, r)
	}
	select {
	case out := <-done:
		if out["status"] != "cancelled" || len(out["steps"].([]any)) != 0 {
			t.Fatal("cancelled turn leaked old result", out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("room close did not cancel inference")
	}
}

func TestAgentDisplaysToolFactsInsteadOfInventedModelPlatforms(t *testing.T) {
	calls := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		message := map[string]any{"role": "assistant", "content": "Watch A on Imaginary Stream; it lasts 999 minutes and guarantees everybody is happy."}
		if calls == 1 {
			message = map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"function": map[string]any{"name": "shortlist", "arguments": map[string]any{"mode": "fair"}}}}}
		}
		json.NewEncoder(w).Encode(map[string]any{"message": message})
	}))
	defer model.Close()
	now := time.Now().UTC()
	h := newHarness(t, workflow.Config{OllamaEndpoint: model.URL, OllamaModel: "controlled", Films: []workflow.Film{{ID: "a", Title: "A", Runtime: 90, Genres: []string{"Comedy"}, Version: "v1"}}, Supplier: &supply{available: true, now: now}})
	h.app.SetMCPEndpoint(h.server.URL + "/mcp")
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 90, "provider_ids": []int{8}, "allow_rewatch": true}})
	status, v := h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", map[string]any{"operation_key": "grounding-0001", "message": "Recommend a movie and explain one candidate using verified facts."})
	reply, _ := v["reply"].(string)
	if status != 200 || strings.Contains(reply, "Imaginary") || strings.Contains(reply, "999") || !strings.Contains(reply, "90 min") {
		t.Fatal("unverified facts displayed", status, v)
	}
	found := false
	for _, step := range v["steps"].([]any) {
		if step.(map[string]any)["tool"] == "explain_candidate" {
			found = true
		}
	}
	if !found {
		t.Fatal("explanation did not traverse actual MCP", v)
	}
}

func TestExplicitLANHostAndSameOriginOnly(t *testing.T) {
	h := newHarness(t, workflow.Config{AllowedHosts: []string{"10.1.2.3"}})
	for _, tc := range []struct {
		host, origin string
		want         int
	}{{"evil.example", "http://evil.example", 403}, {"10.1.2.3", "http://evil.example", 403}, {"10.1.2.3", "http://10.1.2.3", 201}} {
		b, _ := json.Marshal(map[string]any{"name": "LAN", "invite_token": strings.Repeat("7", 64), "operation_key": "create-lan-0001"})
		req, _ := http.NewRequest("POST", h.server.URL+"/api/v1/sessions", bytes.NewReader(b))
		req.Host = tc.host
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Content-Type", "application/json")
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Fatalf("%s %s got %d", tc.host, tc.origin, res.StatusCode)
		}
	}
}

func TestCompetingClientsAndFourMemberCapacity(t *testing.T) {
	h := newHarness(t, workflow.Config{})
	h.cmd("join_member", map[string]any{"name": "A"})
	ready := make(chan struct{})
	type raceResult struct {
		status int
		key    string
	}
	results := make(chan raceResult, 2)
	for i, name := range []string{"B", "C"} {
		go func(i int, name string) {
			<-ready
			status, _ := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", map[string]any{"type": "join_member", "expected_room_revision": 2, "operation_key": fmt.Sprintf("competing-%d", i), "name": name})
			results <- raceResult{status, fmt.Sprintf("competing-%d", i)}
		}(i, name)
	}
	close(ready)
	a, b := <-results, <-results
	if !(a.status == 200 && b.status == 409 || a.status == 409 && b.status == 200) {
		t.Fatal(a, b)
	}
	winner := a.key
	if b.status == 200 {
		winner = b.key
	}
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	h.rev = int(room["room_revision"].(float64))
	h.cmd("join_member", map[string]any{"name": "D"})
	h.cmd("join_member", map[string]any{"name": "E"})
	if status, _ := h.cmd("join_member", map[string]any{"name": "F"}); status != 400 {
		t.Fatal("fifth member allowed", status)
	}
	status, r := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", map[string]any{"type": "join_member", "expected_room_revision": 2, "operation_key": winner, "name": "Different"})
	if status != 409 || r["code"] != "OPERATION_KEY_REUSED" {
		t.Fatal("key reused with different payload", status, r)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSupplier429StopsFurtherRefreshAndLeavesNoCandidates(t *testing.T) {
	now := time.Now().UTC()
	var requestMu sync.Mutex
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestMu.Lock()
		requests++
		requestMu.Unlock()
		status, body := 200, `{"quota":2500,"quotaUsed":1}`
		switch {
		case strings.HasSuffix(r.URL.Path, "/search/"):
			body = `{"title_results":[{"id":10,"type":"movie","imdb_id":"tt0114709"}]}`
		case strings.Contains(r.URL.Path, "/sources/"):
			status = 429
			body = `{"message":"controlled quota fault"}`
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	h := newHarness(t, workflow.Config{Now: func() time.Time { return now }, Films: []workflow.Film{{ID: "a", IMDb: "tt0114709", Title: "A", Runtime: 90, Genres: []string{"Comedy"}, Version: "v1"}}, Supplier: &workflow.Watchmode{Key: "controlled-test-key", Client: client}})
	h.cmd("join_member", map[string]any{"name": "A"})
	h.cmd("join_member", map[string]any{"name": "B"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 90, "provider_ids": []int{203}, "allow_rewatch": true}})
	_, v := h.call("GET", "/api/v1/sessions/"+h.id+"/shortlist", nil)
	if len(v["candidates"].([]any)) != 0 || v["unknown_excluded"] != float64(1) {
		t.Fatal(v)
	}
	before := requests
	now = now.Add(2 * time.Minute)
	_, v = h.call("GET", "/api/v1/sessions/"+h.id+"/shortlist", nil)
	if requests != before || len(v["candidates"].([]any)) != 0 {
		t.Fatal("429 triggered additional HTTP or a false candidate", requests, before, v)
	}
}

func TestConcurrentRetriesRestartAndExpiry(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	db := t.TempDir() + "/state.db"
	h := newHarness(t, workflow.Config{Database: db, Now: func() time.Time { return now }})
	cmd := map[string]any{"type": "join_member", "expected_room_revision": 1, "operation_key": "join-retry-0001", "name": "Alex"}
	var wg sync.WaitGroup
	codes := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s, _ := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", cmd); codes <- s }()
	}
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != 200 {
			t.Fatal(c)
		}
	}
	_, s := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if len(s["members"].([]any)) != 1 || s["room_revision"] != float64(2) {
		t.Fatal(s)
	}
	h.server.Close()
	h.app.Close()
	a, e := workflow.Open(workflow.Config{Database: db, Now: func() time.Time { return now }})
	if e != nil {
		t.Fatal(e)
	}
	h.app = a
	h.server = httptest.NewServer(a.Handler())
	t.Cleanup(func() { h.server.Close(); a.Close() })
	status, rec := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", cmd)
	if status != 200 || rec["replayed"] != true || rec["members"] != nil {
		t.Fatal(status, rec)
	}
	now = now.Add(24 * time.Hour)
	status, s = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if status != 410 || s["members"] != nil {
		t.Fatal(status, s)
	}
	status, _ = h.call("POST", "/api/v1/sessions/"+h.id+"/commands", cmd)
	if status != 410 {
		t.Fatal("expired replay accessible", status)
	}
}

func TestReconnectGetsSnapshotAndCloseRetryReturnsOnlyReceipt(t *testing.T) {
	h := newHarness(t, workflow.Config{})
	h.cmd("join_member", map[string]any{"name": "Alex"})
	req, _ := http.NewRequest("GET", h.server.URL+"/api/v1/sessions/"+h.id+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+h.secret)
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		t.Fatal("events", res.StatusCode)
	}
	b := make([]byte, 256)
	n, e := res.Body.Read(b)
	res.Body.Close()
	if e != nil || !strings.Contains(string(b[:n]), "room_revision\":2") {
		t.Fatal("missing immediate snapshot", string(b[:n]), e)
	}
	cmd := map[string]any{"type": "close_session", "expected_room_revision": h.rev, "operation_key": "close-0001"}
	status, _ := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", cmd)
	if status != 200 {
		t.Fatal(status)
	}
	status, r := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", cmd)
	if status != 200 || r["replayed"] != true || r["members"] != nil {
		t.Fatal(status, r)
	}
	if status, _ = h.call("POST", "/api/v1/sessions", map[string]any{"name": "Tonight", "invite_token": h.secret, "operation_key": "create-0001"}); status != 410 {
		t.Fatal("create replay ignored closed lifecycle", status)
	}
}

type supply struct {
	mu        sync.Mutex
	calls     int
	available bool
	now       time.Time
	onFetch   func(int)
}

func TestReusableHistoryCopiesAndDeletionWithdrawActiveData(t *testing.T) {
	h := newHarness(t, workflow.Config{Films: []workflow.Film{{ID: "film:a", Title: "Example, A", Year: 2000, Runtime: 100, Genres: []string{"Comedy"}, Version: "v1"}}})
	recovery := strings.Repeat("9", 64)
	status, p := h.call("POST", "/api/v1/profiles", map[string]any{"recovery_token": recovery, "operation_key": "profile-0001"})
	if status != 201 {
		t.Fatal(status, p)
	}
	pid := p["resource_id"].(string)
	if status, _ := h.call("GET", "/api/v1/profiles/"+pid, nil); status != 401 {
		t.Fatal("room capability read a profile", status)
	}
	old := h.secret
	h.secret = recovery
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, _ := mw.CreateFormFile("file", "NetflixViewingHistory.csv")
	io.WriteString(part, "\ufeffTitle,Date\n\"Example, A\",10/12/26\n\"Example, A\",10/12/26\nUnknown Title,10/13/26\n")
	mw.WriteField("operation_key", "import-0001")
	mw.WriteField("expected_profile_revision", "1")
	mw.Close()
	req, _ := http.NewRequest("POST", h.server.URL+"/api/v1/profiles/"+pid+"/history-imports", &b)
	req.Header.Set("Authorization", "Bearer "+recovery)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	var report map[string]any
	json.NewDecoder(res.Body).Decode(&report)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.StatusCode, report)
	}
	status, p = h.call("GET", "/api/v1/profiles/"+pid, nil)
	if status != 200 || p["seen_count"] != float64(1) {
		t.Fatal(status, p)
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "10/12/26") || strings.Contains(string(raw), recovery) {
		t.Fatal("raw date or secret leaked")
	}
	h.secret = old
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	if s, r := h.cmd("apply_profile", map[string]any{"member_id": "m1", "profile_id": pid, "recovery_token": recovery}); s != 200 {
		t.Fatal(s, r)
	}
	if s, r := h.cmd("update_member", map[string]any{"member_id": "m1", "patch": map[string]any{"liked_genres": []string{"Comedy"}}}); s != 200 {
		t.Fatal(s, r)
	}
	h.secret = recovery
	_, p = h.call("GET", "/api/v1/profiles/"+pid, nil)
	if len(p["preferences"].(map[string]any)["liked_genres"].([]any)) != 0 {
		t.Fatal("room wrote profile", p)
	}
	status, p = h.call("POST", "/api/v1/profiles/"+pid+"/commands", map[string]any{"type": "update_preferences", "expected_profile_revision": 2, "operation_key": "stable-update-001", "patch": map[string]any{"liked_genres": []string{"Drama"}}})
	if status != 200 {
		t.Fatal(status, p)
	}
	h.secret = old
	_, copyRoom := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	likes := copyRoom["members"].([]any)[0].(map[string]any)["preferences"].(map[string]any)["liked_genres"].([]any)
	if len(likes) != 1 || likes[0] != "Comedy" {
		t.Fatal("profile update implicitly changed room", copyRoom)
	}
	if s, r := h.cmd("apply_profile", map[string]any{"member_id": "m1", "profile_id": pid, "recovery_token": recovery}); s != 200 {
		t.Fatal(s, r)
	}
	_, copyRoom = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	likes = copyRoom["members"].([]any)[0].(map[string]any)["preferences"].(map[string]any)["liked_genres"].([]any)
	if len(likes) != 1 || likes[0] != "Drama" {
		t.Fatal("explicit apply failed", copyRoom)
	}
	h.secret = recovery
	status, p = h.call("POST", "/api/v1/profiles/"+pid+"/commands", map[string]any{"type": "delete_profile", "expected_profile_revision": 3, "operation_key": "delete-0001"})
	if status != 200 {
		t.Fatal(status, p)
	}
	if status, _ = h.call("GET", "/api/v1/profiles/"+pid, nil); status != 410 {
		t.Fatal("deleted profile accessible", status)
	}
	h.secret = old
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	m := room["members"].([]any)[0].(map[string]any)
	if m["source_profile_id"] != nil || m["seen_count"] != float64(0) {
		t.Fatal("profile copy retained", m)
	}
	h.secret = recovery
	status, _ = h.call("POST", "/api/v1/profiles/"+pid+"/commands", map[string]any{"type": "update_preferences", "expected_profile_revision": 1, "operation_key": "import-0001"})
	if status != 410 {
		t.Fatal("replay bypassed tombstone", status)
	}
	if status, _ = h.call("POST", "/api/v1/profiles", map[string]any{"recovery_token": recovery, "operation_key": "profile-0001"}); status != 410 {
		t.Fatal("create replay ignored deleted profile", status)
	}
}

func (s *supply) Fetch(_ context.Context, f workflow.Film) (workflow.Availability, error) {
	s.mu.Lock()
	s.calls++
	n := s.calls
	s.mu.Unlock()
	if s.onFetch != nil {
		s.onFetch(n)
	}
	offers := []workflow.Offer{}
	if s.available {
		offers = append(offers, workflow.Offer{Region: "US", ProviderID: 8, Monetization: "sub"})
	}
	return workflow.Availability{FilmID: f.ID, Known: true, Offers: offers, FetchedAt: s.now, ExpiresAt: s.now.Add(time.Hour)}, nil
}

type supplierFunc func(context.Context, workflow.Film) (workflow.Availability, error)

func (f supplierFunc) Fetch(ctx context.Context, m workflow.Film) (workflow.Availability, error) {
	return f(ctx, m)
}
func TestUnknownFactsAndWrongMonetizationNeverQualify(t *testing.T) {
	now := time.Now().UTC()
	films := []workflow.Film{}
	for _, id := range []string{"valid", "unknown-runtime", "conflicting-runtime", "too-long", "genre-veto", "seen", "rent", "buy", "wrong-region", "unknown-supply", "expired"} {
		films = append(films, workflow.Film{ID: id, Title: id, Runtime: 100, Genres: []string{"Comedy"}, Version: "v1", Prior: 3})
	}
	films[1].Runtime = 0
	films[2].RuntimeConflict = true
	films[3].Runtime = 101
	films[4].Genres = []string{"Horror"}
	h := newHarness(t, workflow.Config{Now: func() time.Time { return now }, Films: films, Supplier: supplierFunc(func(ctx context.Context, f workflow.Film) (workflow.Availability, error) {
		av := workflow.Availability{Known: true, ExpiresAt: now.Add(time.Hour), FetchedAt: now, Offers: []workflow.Offer{{Region: "US", ProviderID: 8, Monetization: "sub"}}}
		switch f.ID {
		case "rent", "buy":
			av.Offers[0].Monetization = f.ID
		case "wrong-region":
			av.Offers[0].Region = "CA"
		case "unknown-supply":
			return av, fmt.Errorf("controlled supplier timeout")
		case "expired":
			av.ExpiresAt = now.Add(-time.Hour)
		}
		return av, nil
	})})
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 100, "provider_ids": []int{8}, "allow_rewatch": false}})
	h.cmd("update_member", map[string]any{"member_id": "m1", "patch": map[string]any{"vetoes": []string{"Horror"}, "seen": []string{"seen"}}})
	for _, mode := range []string{"fair", "average", "catalog_prior"} {
		status, list := h.call("GET", "/api/v1/sessions/"+h.id+"/shortlist?mode="+mode, nil)
		if status != 200 {
			t.Fatal(status, list)
		}
		cs := list["candidates"].([]any)
		if len(cs) != 1 || cs[0].(map[string]any)["film"].(map[string]any)["film_id"] != "valid" {
			t.Fatal("hard conditions changed with ranking", mode, list)
		}
		if cs[0].(map[string]any)["estimated_group_score"] != float64(3) {
			t.Fatal("seen became like", cs)
		}
	}
}

func TestImportRejectsUnknownHeadersAndPreservesManualInputsOnReplace(t *testing.T) {
	h := newHarness(t, workflow.Config{Films: []workflow.Film{{ID: "a", Title: "Twin", Year: 1990, Genres: []string{"Drama"}, Runtime: 90}, {ID: "b", Title: "Twin", Year: 2000, Genres: []string{"Drama"}, Runtime: 95}, {ID: "c", Title: "A: Movie", Year: 2001, Genres: []string{"Drama"}, Runtime: 90}}, KnownEpisodes: []string{"Example Show: Verified Episode"}})
	h.cmd("join_member", map[string]any{"name": "A"})
	h.cmd("update_member", map[string]any{"member_id": "m1", "patch": map[string]any{"seen": []string{"a"}, "ratings": map[string]float64{"a": 4}}})
	upload := func(csvText, col, mode, key string) int {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		p, _ := mw.CreateFormFile("file", "history.csv")
		io.WriteString(p, csvText)
		for k, v := range map[string]string{"operation_key": key, "expected_room_revision": fmt.Sprint(h.rev), "target_member_id": "m1", "title_column": col, "mode": mode} {
			mw.WriteField(k, v)
		}
		mw.Close()
		req, _ := http.NewRequest("POST", h.server.URL+"/api/v1/sessions/"+h.id+"/history-imports", &b)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+h.secret)
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var v map[string]any
		json.NewDecoder(res.Body).Decode(&v)
		if res.StatusCode == 200 {
			h.rev = int(v["room_revision"].(float64))
		}
		return res.StatusCode
	}
	if s := upload("Movie Name,Date\nTwin,unknown\n", "", "append", "import-header-01"); s != 400 {
		t.Fatal(s)
	}
	if s := upload("Movie Name,Date\nTwin,unknown\nA: Movie,unknown\nExample Show: Verified Episode,unknown\nA: Movie,unknown\n", "Movie Name", "append", "import-boundary-01"); s != 200 {
		t.Fatal(s)
	}
	_, v := h.call("GET", "/api/v1/sessions/"+h.id+"/members/m1/history", nil)
	report := v["report"].(map[string]any)
	if report["total_rows"] != float64(4) || report["duplicate"] != float64(1) || report["ambiguous"] != float64(1) || report["matched_movie"] != float64(1) || report["known_episode"] != float64(1) {
		t.Fatal(v)
	}
	previousRevision := h.rev
	if status := upload("Movie Name,Date\nTwin,unknown\nA: Movie,unknown\nExample Show: Verified Episode,unknown\nA: Movie,unknown\n", "Movie Name", "append", "import-identical-02"); status != 200 || h.rev != previousRevision {
		t.Fatal("identical import changed the decision", status, h.rev, previousRevision)
	}
	if s := upload("Title,Date\nTwin (2000),unknown\n", "", "replace", "import-replace-01"); s != 200 {
		t.Fatal(s)
	}
	_, v = h.call("GET", "/api/v1/sessions/"+h.id+"/members/m1/history", nil)
	if v["seen_total"] != float64(2) {
		t.Fatal("replace removed manual or retained old import", v)
	}
	_, v = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	prefs := v["members"].([]any)[0].(map[string]any)["preferences"].(map[string]any)
	if prefs["ratings"].(map[string]any)["a"] != float64(4) {
		t.Fatal("replace removed manual rating")
	}
	if s := upload("Title\n"+strings.Repeat("x", 8193)+"\n", "", "append", "import-oversize-01"); s != 413 {
		t.Fatal("oversized field", s)
	}
	if s := upload("Title\n"+strings.Repeat("unknown\n", 50001), "", "append", "import-row-cap-01"); s != 413 {
		t.Fatal("row cap", s)
	}
	if s := upload("Title\n"+strings.Repeat("x", (5<<20)+1), "", "append", "import-file-cap-01"); s != 413 {
		t.Fatal("file cap", s)
	}
	if s := upload("Title\n\xff\n", "", "append", "import-utf8-01"); s != 400 {
		t.Fatal("invalid UTF-8", s)
	}
	if s := upload("Title,Date\n\"A:\n Movie\",unknown\n", "", "append", "import-newline-01"); s != 200 {
		t.Fatal("quoted newline", s)
	}
	_, v = h.call("GET", "/api/v1/sessions/"+h.id+"/members/m1/history", nil)
	if v["report"].(map[string]any)["matched_movie"] != float64(1) {
		t.Fatal("newline title did not match", v)
	}
}

func TestDeletingLinkedProfileCancelsModelAndWithdrawsHistory(t *testing.T) {
	entered := make(chan struct{})
	privateID := "fixture-private-seen-marker-482179"
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if bytes.Contains(b, []byte(privateID)) {
			t.Error("full watched history was sent to the model")
		}
		close(entered)
		<-r.Context().Done()
	}))
	defer model.Close()
	h := newHarness(t, workflow.Config{OllamaEndpoint: model.URL, OllamaModel: "controlled", Films: []workflow.Film{{ID: privateID, Title: "A", Runtime: 90, Genres: []string{"Drama"}}}})
	h.app.SetMCPEndpoint(h.server.URL + "/mcp")
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	recovery := strings.Repeat("9", 64)
	_, p := h.call("POST", "/api/v1/profiles", map[string]any{"recovery_token": recovery, "operation_key": "create-private-01"})
	pid := p["resource_id"].(string)
	roomSecret := h.secret
	h.secret = recovery
	h.call("POST", "/api/v1/profiles/"+pid+"/commands", map[string]any{"type": "update_preferences", "expected_profile_revision": 1, "operation_key": "update-private-01", "patch": map[string]any{"seen_add": []string{privateID}}})
	h.secret = roomSecret
	h.cmd("apply_profile", map[string]any{"member_id": "m1", "profile_id": pid, "recovery_token": recovery})
	done := make(chan map[string]any, 1)
	go func() {
		_, out := h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", map[string]any{"message": "Recommend a movie", "operation_key": "private-agent-01"})
		done <- out
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("model did not start")
	}
	h.secret = recovery
	status, v := h.call("POST", "/api/v1/profiles/"+pid+"/commands", map[string]any{"type": "delete_profile", "expected_profile_revision": 2, "operation_key": "delete-private-01"})
	h.secret = roomSecret
	if status != 200 {
		t.Fatal(status, v)
	}
	select {
	case out := <-done:
		if out["status"] != "cancelled" {
			t.Fatal(out)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("profile deletion did not cancel model")
	}
	_, v = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if v["members"].([]any)[0].(map[string]any)["seen_count"] != float64(0) {
		t.Fatal("history copy retained", v)
	}
}

func TestSavingRejectsChangedOrUnknownFreshSupply(t *testing.T) {
	for _, mode := range []string{"unknown", "changed"} {
		t.Run(mode, func(t *testing.T) {
			changed := false
			now := time.Now().UTC()
			h := newHarness(t, workflow.Config{Films: []workflow.Film{{ID: "a", Title: "A", Runtime: 90, Genres: []string{"Drama"}, Version: "v1"}}, Supplier: supplierFunc(func(ctx context.Context, f workflow.Film) (workflow.Availability, error) {
				av := workflow.Availability{Known: true, FetchedAt: now, ExpiresAt: now.Add(time.Hour), Offers: []workflow.Offer{{Region: "US", ProviderID: 8, Monetization: "sub"}}}
				if changed {
					if mode == "unknown" {
						return av, fmt.Errorf("controlled network timeout")
					}
					av.Offers = []workflow.Offer{}
				}
				return av, nil
			})})
			h.cmd("join_member", map[string]any{"name": "A"})
			h.cmd("join_member", map[string]any{"name": "B"})
			h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 90, "provider_ids": []int{8}, "allow_rewatch": true}})
			h.cmd("propose_candidate", map[string]any{"film_id": "a"})
			_, s := h.call("GET", "/api/v1/sessions/"+h.id, nil)
			for _, m := range s["members"].([]any) {
				h.cmd("record_acceptance", map[string]any{"target_member_id": m.(map[string]any)["id"], "film_id": "a", "decision_version": s["decision_version"], "response": "accept"})
			}
			changed = true
			status, v := h.cmd("save_choice", map[string]any{"film_id": "a", "decision_version": s["decision_version"]})
			want := 409
			if mode == "unknown" {
				want = 503
			}
			if status != want {
				t.Fatal(status, v)
			}
			_, s = h.call("GET", "/api/v1/sessions/"+h.id, nil)
			if s["active_candidate"] != nil || s["last_choice"] != nil {
				t.Fatal("invalid supply preserved/saved choice", s)
			}
		})
	}
}

func TestSaveRefreshDoesNotLockRoomAndRejectsConcurrentPolicyChange(t *testing.T) {
	now := time.Now().UTC()
	entered, release := make(chan struct{}), make(chan struct{})
	supplier := &supply{available: true, now: now, onFetch: func(n int) {
		if n == 2 {
			close(entered)
			<-release
		}
	}}
	h := newHarness(t, workflow.Config{Films: []workflow.Film{{ID: "film:a", Title: "A", Runtime: 100, Genres: []string{"Comedy"}, Version: "v1"}}, Supplier: supplier})
	h.cmd("join_member", map[string]any{"name": "A"})
	h.cmd("join_member", map[string]any{"name": "B"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 100, "provider_ids": []int{8}, "allow_rewatch": true}})
	h.cmd("propose_candidate", map[string]any{"film_id": "film:a"})
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	dv := room["decision_version"]
	for _, m := range room["members"].([]any) {
		h.cmd("record_acceptance", map[string]any{"target_member_id": m.(map[string]any)["id"], "film_id": "film:a", "decision_version": dv, "response": "accept"})
	}
	done := make(chan int, 1)
	rev := h.rev
	go func() {
		status, _ := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", map[string]any{"type": "save_choice", "expected_room_revision": rev, "operation_key": "race-save-0001", "film_id": "film:a", "decision_version": dv})
		done <- status
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("refresh did not start")
	}
	status, r := h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 99, "provider_ids": []int{8}, "allow_rewatch": true}})
	close(release)
	if status != 200 {
		t.Fatal("network refresh blocked room mutation", status, r)
	}
	if status = <-done; status != 409 {
		t.Fatal("stale save committed", status)
	}
}

type harness struct {
	t          *testing.T
	app        *workflow.App
	server     *httptest.Server
	secret, id string
	rev        int
}

func newHarness(t *testing.T, cfg workflow.Config) *harness {
	t.Helper()
	if cfg.Database == "" {
		cfg.Database = t.TempDir() + "/workflow.db"
	}
	a, e := workflow.Open(cfg)
	if e != nil {
		t.Fatal(e)
	}
	h := &harness{t: t, app: a, server: httptest.NewServer(a.Handler()), secret: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	t.Cleanup(func() { h.server.Close(); a.Close() })
	_, r := h.call("POST", "/api/v1/sessions", map[string]any{"name": "Tonight", "invite_token": h.secret, "operation_key": "create-0001"})
	h.id = r["resource_id"].(string)
	h.rev = 1
	return h
}
func (h *harness) call(method, path string, body any) (int, map[string]any) {
	h.t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, h.server.URL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+h.secret)
	req.Header.Set("Content-Type", "application/json")
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		h.t.Fatal(e)
	}
	defer res.Body.Close()
	var v map[string]any
	json.NewDecoder(res.Body).Decode(&v)
	return res.StatusCode, v
}
func (h *harness) cmd(kind string, extra map[string]any) (int, map[string]any) {
	h.t.Helper()
	if extra == nil {
		extra = map[string]any{}
	}
	extra["type"] = kind
	extra["operation_key"] = fmt.Sprintf("command-%d-%s", h.rev, kind)
	extra["expected_room_revision"] = h.rev
	status, r := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", extra)
	if status == 200 {
		h.rev = int(r["room_revision"].(float64))
	}
	return status, r
}

func TestGroupChoiceHonorsHardConditionsAndVersionedAcceptances(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	supplier := &supply{available: true, now: now}
	h := newHarness(t, workflow.Config{Now: func() time.Time { return now }, Films: []workflow.Film{{ID: "film:a", Title: "Example A", Runtime: 100, Genres: []string{"Comedy"}, Version: "v1"}, {ID: "film:b", Title: "Example B", Runtime: 101, Genres: []string{"Horror"}, Version: "v1"}}, Supplier: supplier})
	for _, name := range []string{"Alex", "Sam"} {
		if s, r := h.cmd("join_member", map[string]any{"name": name}); s != 200 {
			t.Fatal(s, r)
		}
	}
	if s, r := h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 100, "provider_ids": []int{8}, "allow_rewatch": false}}); s != 200 {
		t.Fatal(s, r)
	}
	s, list := h.call("GET", "/api/v1/sessions/"+h.id+"/shortlist", nil)
	if s != 200 || len(list["candidates"].([]any)) != 1 {
		t.Fatal(s, list)
	}
	if s, r := h.cmd("propose_candidate", map[string]any{"film_id": "film:a"}); s != 200 {
		t.Fatal(s, r)
	}
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	dv := room["decision_version"]
	members := room["members"].([]any)
	for i, m := range members {
		if s, r := h.cmd("record_acceptance", map[string]any{"target_member_id": m.(map[string]any)["id"], "film_id": "film:a", "decision_version": dv, "response": "accept", "actor_member_id": "m1"}); s != 200 || r["decision_version"] != dv {
			t.Fatal(s, r)
		}
		if i == 0 {
			if s, _ := h.cmd("save_choice", map[string]any{"film_id": "film:a", "decision_version": dv}); s != 409 {
				t.Fatal("incomplete saved", s)
			}
		}
	}
	before := supplier.calls
	save := map[string]any{"type": "save_choice", "operation_key": "save-0001", "expected_room_revision": h.rev, "film_id": "film:a", "decision_version": dv}
	s, receipt := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", save)
	if s != 200 || receipt["summary_id"] == nil {
		t.Fatal(s, receipt)
	}
	h.rev = int(receipt["room_revision"].(float64))
	s, r := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", save)
	if s != 200 || r["replayed"] != true || supplier.calls != before+1 {
		t.Fatal("replay refreshed or duplicate", s, r, supplier.calls, before)
	}
	s, r = h.cmd("save_choice", map[string]any{"film_id": "film:a", "decision_version": dv})
	if s != 200 || r["summary_id"] != receipt["summary_id"] || supplier.calls != before+1 {
		t.Fatal("same decision saved twice or refreshed", s, r)
	}
	if s, r := h.cmd("update_member", map[string]any{"member_id": "m1", "patch": map[string]any{"vetoes": []string{"Comedy"}, "liked_genres": []string{"Comedy"}}}); s != 200 {
		t.Fatal(s, r)
	}
	_, room = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if room["active_candidate"] != nil || len(room["acceptances"].(map[string]any)) != 0 {
		t.Fatal("old acceptance survived", room)
	}
	_, list = h.call("GET", "/api/v1/sessions/"+h.id+"/shortlist", nil)
	if len(list["candidates"].([]any)) != 0 {
		t.Fatal("hard conditions relaxed", list)
	}
}

func TestExpiredCandidateFactsClearRecordedResponsesOnRead(t *testing.T) {
	now := time.Now().UTC()
	supplier := &supply{available: true, now: now}
	h := newHarness(t, workflow.Config{Now: func() time.Time { return now }, Films: []workflow.Film{{ID: "a", Title: "A", Runtime: 90, Genres: []string{"Drama"}, Version: "v1"}}, Supplier: supplier})
	h.cmd("join_member", map[string]any{"name": "A"})
	h.cmd("join_member", map[string]any{"name": "B"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 90, "provider_ids": []int{8}, "allow_rewatch": true}})
	h.cmd("propose_candidate", map[string]any{"film_id": "a"})
	_, s := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	h.cmd("record_acceptance", map[string]any{"target_member_id": "m1", "film_id": "a", "decision_version": s["decision_version"], "response": "accept"})
	now = now.Add(2 * time.Hour)
	_, s = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if s["active_candidate"] != nil || len(s["acceptances"].(map[string]any)) != 0 {
		t.Fatal("expired facts kept responses", s)
	}
}

func TestSessionCapabilityAndCompetingWrites(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	app, err := workflow.Open(workflow.Config{Database: t.TempDir() + "/workflow.db", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	call := func(path, token string, body any) (int, map[string]any) {
		t.Helper()
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", server.URL+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		out := map[string]any{}
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	status, created := call("/api/v1/sessions", "", map[string]any{"name": "Tonight", "invite_token": token, "operation_key": "create-0001"})
	if status != 201 {
		t.Fatalf("create %d %v", status, created)
	}
	id := created["resource_id"].(string)
	status, _ = call("/api/v1/sessions/"+id+"/commands", "wrong", map[string]any{"type": "join_member", "expected_room_revision": 1, "operation_key": "join-0001", "name": "Alex"})
	if status != 401 {
		t.Fatalf("wrong capability %d", status)
	}
	cmd := map[string]any{"type": "join_member", "expected_room_revision": 1, "operation_key": "join-0001", "name": "Alex"}
	status, receipt := call("/api/v1/sessions/"+id+"/commands", token, cmd)
	if status != 200 || receipt["room_revision"] != float64(2) {
		t.Fatalf("join %d %v", status, receipt)
	}
	status, replayed := call("/api/v1/sessions/"+id+"/commands", token, cmd)
	if status != 200 || replayed["replayed"] != true {
		t.Fatalf("retry %d %v", status, replayed)
	}
	status, _ = call("/api/v1/sessions/"+id+"/commands", token, map[string]any{"type": "join_member", "expected_room_revision": 1, "operation_key": "join-0002", "name": "Sam"})
	if status != 409 {
		t.Fatalf("stale join %d", status)
	}
}
