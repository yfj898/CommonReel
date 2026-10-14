package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"commonreel/internal/catalog"
	"commonreel/internal/recommend"
	"commonreel/internal/room"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func decoded[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	var out T
	b, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestStreamableHTTPWorkflow(t *testing.T) {
	s, err := room.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := New(s, catalog.Fixture())
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", DisableStandaloneSSE: true}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if cs.InitializeResult().ProtocolVersion != "2025-11-25" {
		t.Fatal("wrong protocol")
	}
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 9 {
		t.Fatalf("tool discovery %v %v", tools, err)
	}
	call := func(name string, args any) *mcp.CallToolResult {
		t.Helper()
		res, e := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e != nil {
			t.Fatal(e)
		}
		if res.IsError {
			t.Fatalf("tool %s failed: %v", name, res.Content)
		}
		return res
	}
	r := decoded[room.Room](t, call("create_room", CreateInput{Name: "MCP night"}))
	r = decoded[room.Room](t, call("set_member", MemberInput{RoomID: r.ID, Revision: r.Revision, OperationKey: "join-key-0001", Name: "Alex", Profile: 1, Likes: []string{"Comedy"}, Vetoes: []string{"Horror"}}))
	r = decoded[room.Room](t, call("set_policy", PolicyInput{RoomID: r.ID, Revision: r.Revision, OperationKey: "policy-key-0001", ExcludeSeen: false}))
	list := decoded[recommend.Result](t, call("shortlist", ShortlistInput{RoomID: r.ID, Limit: 6, Mode: "fair"}))
	if len(list.Candidates) == 0 {
		t.Fatal("empty")
	}
	input := SaveInput{RoomID: r.ID, Revision: r.Revision, OperationKey: "save-key-0001", MovieID: list.Candidates[0].Movie.ID}
	r = decoded[room.Room](t, call("save_choice", input))
	replayed := decoded[room.Room](t, call("save_choice", input))
	if len(r.Choices) != 1 || replayed.Revision != r.Revision {
		t.Fatal("save retry duplicated")
	}
	stale, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "save_choice", Arguments: SaveInput{RoomID: r.ID, Revision: 1, OperationKey: "save-key-stale", MovieID: 1}})
	if err != nil || !stale.IsError {
		t.Fatalf("stale save not rejected: %v", err)
	}
	read := decoded[room.Room](t, call("get_room", RoomInput{RoomID: r.ID}))
	if read.Choices[0].FromRevision != list.RoomRevision {
		t.Fatal("receipt source revision wrong")
	}
}
func TestCrossOriginAndHostProtection(t *testing.T) {
	s, _ := room.Open(":memory:")
	defer s.Close()
	a := New(s, catalog.Fixture())
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	req, _ := http.NewRequest("POST", server.URL+"/mcp", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://unrelated.example")
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("cross origin status %d", res.StatusCode)
	}
	req, _ = http.NewRequest("GET", server.URL+"/health", nil)
	req.Host = "attacker.example"
	res, err = server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatal("rebound host accepted")
	}
}
