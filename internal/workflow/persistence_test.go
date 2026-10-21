package workflow_test

import (
	"commonreel/internal/workflow"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgentReceiptFailureIsReportedAndCanBeRetried(t *testing.T) {
	calls := 0
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "qwen3.5:4b"}}})
			return
		}
		calls++
		var in struct {
			Model    string            `json:"model"`
			Messages []json.RawMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Error(err)
		}
		if in.Model != "qwen3.5:4b" || len(in.Messages) != 2 {
			t.Errorf("wrong default or uncommitted conversation reused: %+v", in)
		}
		json.NewEncoder(w).Encode(map[string]any{"message": map[string]any{"role": "assistant", "content": "Hello."}})
	}))
	defer model.Close()
	path := t.TempDir() + "/workflow.db"
	h := newHarness(t, workflow.Config{Database: path, OllamaEndpoint: model.URL})
	if err := h.app.SetMCPEndpoint(h.server.URL + "/mcp"); err != nil {
		t.Fatal(err)
	}
	_, health := h.call("GET", "/health", nil)
	if m := health["model"].(map[string]any); m["name"] != "qwen3.5:4b" || m["status"] != "installed; tool performance requires separate evidence" {
		t.Fatal("health did not use the actual model default", health)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER reject_agent_receipt BEFORE INSERT ON v2_operations WHEN NEW.key LIKE 'agent-%' BEGIN SELECT RAISE(FAIL, 'controlled receipt write failure'); END`); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"operation_key": "receipt-0001", "message": "Hello"}
	status, out := h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", input)
	if status != 500 || out["code"] != "INTERNAL_ERROR" {
		t.Fatal("receipt failure was returned as success", status, out)
	}
	if _, err = db.Exec("DROP TRIGGER reject_agent_receipt"); err != nil {
		t.Fatal(err)
	}
	status, out = h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", input)
	if status != 200 || out["status"] != "completed" || out["model"] != "qwen3.5:4b" {
		t.Fatal("retry after failed receipt", status, out)
	}
	status, out = h.call("POST", "/api/v1/sessions/"+h.id+"/agent-turns", input)
	if status != 200 || out["status"] != "already_processed" || calls != 2 {
		t.Fatal("committed turn did not replay", status, out, calls)
	}
}

func TestStoredSessionReadFailuresReachSweepAndOpen(t *testing.T) {
	path := t.TempDir() + "/workflow.db"
	h := newHarness(t, workflow.Config{Database: path})
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO v2_sessions VALUES(?, ?)", "damaged-session", "{"); err != nil {
		t.Fatal(err)
	}
	var syntax *json.SyntaxError
	if err = h.app.Sweep(context.Background()); !errors.As(err, &syntax) {
		t.Fatal("sweep hid the stored-data error", err)
	}
	app, err := workflow.Open(workflow.Config{Database: path})
	if app != nil {
		app.Close()
		t.Fatal("open accepted unreadable session data")
	}
	if !errors.As(err, &syntax) {
		t.Fatal("open lost the stored-data error", err)
	}
}

func TestRepeatingProposalKeepsResponsesAndNewProposalClearsThem(t *testing.T) {
	now := time.Now().UTC()
	h := newHarness(t, workflow.Config{Films: []workflow.Film{
		{ID: "a", Title: "A", Runtime: 90, Genres: []string{"Comedy"}, Version: "v1"},
		{ID: "b", Title: "B", Runtime: 90, Genres: []string{"Comedy"}, Version: "v1"},
	}, Supplier: &supply{available: true, now: now}})
	h.cmd("join_member", map[string]any{"name": "Alex"})
	h.cmd("join_member", map[string]any{"name": "Sam"})
	h.cmd("update_policy", map[string]any{"policy": map[string]any{"max_runtime_minutes": 90, "provider_ids": []int{8}, "allow_rewatch": true}})
	if status, result := h.cmd("propose_candidate", map[string]any{"film_id": "a"}); status != 200 {
		t.Fatal(status, result)
	}
	_, room := h.call("GET", "/api/v1/sessions/"+h.id, nil)
	decision := room["decision_version"].(float64)
	for _, raw := range room["members"].([]any) {
		if status, result := h.cmd("record_acceptance", map[string]any{"target_member_id": raw.(map[string]any)["id"], "film_id": "a", "decision_version": decision, "response": "accept"}); status != 200 || result["decision_version"] != decision {
			t.Fatal(status, result)
		}
	}
	revision := h.rev
	if status, result := h.cmd("propose_candidate", map[string]any{"film_id": "a"}); status != 200 || result["code"] != "no_change" {
		t.Fatal(status, result)
	}
	_, room = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	if h.rev != revision || room["decision_version"] != decision || len(room["acceptances"].(map[string]any)) != 2 {
		t.Fatal("same proposal invalidated responses", room)
	}
	if status, result := h.call("POST", "/api/v1/sessions/"+h.id+"/commands", map[string]any{"type": "propose_candidate", "film_id": "b", "expected_room_revision": h.rev, "operation_key": "new-proposal-0002"}); status != 200 {
		t.Fatal(status, result)
	}
	_, room = h.call("GET", "/api/v1/sessions/"+h.id, nil)
	candidate := room["active_candidate"].(map[string]any)
	if room["room_revision"] != float64(revision+1) || room["decision_version"] != decision+1 || candidate["film_id"] != "b" || candidate["decision_version"] != decision+1 || len(room["acceptances"].(map[string]any)) != 0 {
		t.Fatal("new proposal did not advance the decision", room)
	}
}
