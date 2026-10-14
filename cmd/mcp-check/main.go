package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"commonreel/internal/app"
	"commonreel/internal/recommend"
	"commonreel/internal/room"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func decode[T any](r *mcp.CallToolResult) T {
	var out T
	b, e := json.Marshal(r.StructuredContent)
	if e != nil {
		log.Fatal(e)
	}
	if e = json.Unmarshal(b, &out); e != nil {
		log.Fatal(e)
	}
	return out
}
func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:8092/mcp", "local endpoint")
	output := flag.String("out", "docs/evidence/mcp-live.json", "probe report")
	flag.Parse()
	u, err := url.Parse(*endpoint)
	if err != nil {
		log.Fatal(err)
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || ip == nil || !ip.IsLoopback() {
		log.Fatal("probe is limited to loopback HTTP")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "CommonReel live probe", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: *endpoint, DisableStandaloneSSE: true}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	if session.InitializeResult().ProtocolVersion != "2025-11-25" {
		log.Fatal("unexpected protocol")
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	names := []string{}
	for _, t := range listed.Tools {
		names = append(names, t.Name)
	}
	call := func(name string, input any) *mcp.CallToolResult {
		r, e := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: input})
		if e != nil {
			log.Fatal(e)
		}
		if r.IsError {
			log.Fatalf("%s failed: %v", name, r.Content)
		}
		return r
	}
	info := decode[app.Info](call("catalog_info", app.Empty{}))
	if len(info.Profiles) < 3 {
		log.Fatal("insufficient training profiles")
	}
	r := decode[room.Room](call("create_room", app.CreateInput{Name: "MCP live verification"}))
	for i, name := range []string{"Probe Alex", "Probe Jamie", "Probe Sam"} {
		profile := int(info.Profiles[i]["id"].(float64))
		vetoes := []string{}
		if i == 1 {
			vetoes = []string{"Horror"}
		}
		r = decode[room.Room](call("set_member", app.MemberInput{RoomID: r.ID, Revision: r.Revision, OperationKey: room.ID(), Name: name, Profile: profile, Likes: []string{}, Vetoes: vetoes}))
	}
	start := time.Now()
	result := decode[recommend.Result](call("shortlist", app.ShortlistInput{RoomID: r.ID, Mode: "fair", Limit: 6}))
	mcpMS := float64(time.Since(start).Microseconds()) / 1000
	if len(result.Candidates) == 0 {
		log.Fatal("empty shortlist")
	}
	violations := 0
	for _, c := range result.Candidates {
		for _, g := range c.Movie.Genres {
			if g == "Horror" {
				violations++
			}
		}
	}
	if violations > 0 {
		log.Fatal("veto violation")
	}
	input := app.SaveInput{RoomID: r.ID, Revision: r.Revision, OperationKey: room.ID(), MovieID: result.Candidates[0].Movie.ID}
	r = decode[room.Room](call("save_choice", input))
	replay := decode[room.Room](call("save_choice", input))
	if len(replay.Choices) != 1 || replay.Revision != r.Revision {
		log.Fatal("idempotency failure")
	}
	stale, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "save_choice", Arguments: app.SaveInput{RoomID: r.ID, Revision: 1, OperationKey: room.ID(), MovieID: input.MovieID}})
	if e != nil || !stale.IsError {
		log.Fatal("stale save was not rejected")
	}
	read := decode[room.Room](call("get_room", app.RoomInput{RoomID: r.ID}))
	if read.Revision != r.Revision || len(read.Choices) != 1 {
		log.Fatal("readback mismatch")
	}
	report := map[string]any{"run_utc": time.Now().UTC().Format(time.RFC3339), "endpoint": *endpoint, "protocol": session.InitializeResult().ProtocolVersion, "transport": "actual Streamable HTTP via official Go MCP SDK", "tools": names, "dataset": info.Data, "workflow": "initialize -> tools/list -> catalog_info -> create_room -> 3 set_member -> shortlist -> save_choice -> identical retry -> stale rejection -> get_room", "room_revision": read.Revision, "choice_count_after_retry": len(read.Choices), "stale_choice_rejected": true, "veto_violations": violations, "shortlist_count": len(result.Candidates), "one_shortlist_round_trip_ms": mcpMS, "limits": "one local HTTP workflow on real data, constructed group; not production load, Alexa device or human study"}
	b, _ := json.MarshalIndent(report, "", "  ")
	if e = os.MkdirAll(filepath.Dir(*output), 0755); e != nil {
		log.Fatal(e)
	}
	if e = os.WriteFile(*output, b, 0644); e != nil {
		log.Fatal(e)
	}
	fmt.Println(string(b))
}
