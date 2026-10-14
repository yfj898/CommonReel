package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"commonreel/internal/catalog"
	"commonreel/internal/recommend"
	"commonreel/internal/room"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed web/*
var assets embed.FS

type App struct {
	Store  *room.Store
	Engine *recommend.Engine
	MCP    *mcp.Server
	hub    *hub
}
type hub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan int]bool
}

func (h *hub) publish(id string, rev int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers[id] {
		select {
		case ch <- rev:
		default:
		}
	}
}
func New(store *room.Store, c *catalog.Catalog) *App {
	a := &App{Store: store, Engine: recommend.New(c), hub: &hub{subscribers: map[string]map[chan int]bool{}}}
	a.MCP = mcp.NewServer(&mcp.Implementation{Name: "CommonReel", Version: "0.1.0"}, &mcp.ServerOptions{Instructions: "Help a group choose a film. Ask for preferences, respect every veto, and explain each member's predicted enjoyment. Ratings are estimates, not human satisfaction measurements. Never claim playback rights or streaming availability. Read the current room revision before mutations; reuse an operation key only for an identical retry. Save a choice only after the user chooses it.", SupportedProtocolVersions: []string{"2025-11-25"}})
	a.register()
	return a
}
func tool(name, desc string, read bool) *mcp.Tool {
	closed := false
	return &mcp.Tool{Name: name, Description: desc, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: read, DestructiveHint: &closed, OpenWorldHint: &closed, IdempotentHint: name != "create_room"}}
}

type Empty struct{}
type Info struct {
	Data      catalog.Metadata `json:"data"`
	Profiles  []map[string]any `json:"profiles"`
	Genres    []string         `json:"genres"`
	Protocol  string           `json:"protocol"`
	Assistant string           `json:"assistant"`
}
type CreateInput struct {
	Name string `json:"name"`
}
type RoomInput struct {
	RoomID string `json:"room_id"`
}
type MemberInput struct {
	RoomID       string   `json:"room_id"`
	Revision     int      `json:"revision"`
	OperationKey string   `json:"operation_key"`
	MemberID     string   `json:"member_id"`
	Name         string   `json:"name"`
	Profile      int      `json:"profile"`
	Likes        []string `json:"likes"`
	Vetoes       []string `json:"vetoes"`
}
type PolicyInput struct {
	RoomID       string `json:"room_id"`
	Revision     int    `json:"revision"`
	OperationKey string `json:"operation_key"`
	ExcludeSeen  bool   `json:"exclude_seen"`
}
type ShortlistInput struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit"`
	Mode   string `json:"mode"`
}
type SaveInput struct {
	RoomID       string `json:"room_id"`
	Revision     int    `json:"revision"`
	OperationKey string `json:"operation_key"`
	MovieID      int    `json:"movie_id"`
}
type SearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}
type SearchOutput struct {
	Movies []catalog.Movie `json:"movies"`
}
type RemoveInput struct {
	RoomID       string `json:"room_id"`
	Revision     int    `json:"revision"`
	OperationKey string `json:"operation_key"`
	MemberID     string `json:"member_id"`
}

func (a *App) register() {
	mcp.AddTool(a.MCP, tool("catalog_info", "Read dataset provenance, available profile IDs and genres. No demographic data is loaded.", true), func(ctx context.Context, req *mcp.CallToolRequest, in Empty) (*mcp.CallToolResult, Info, error) {
		genres := map[string]bool{}
		for _, m := range a.Engine.Catalog.Movies {
			for _, g := range m.Genres {
				genres[g] = true
			}
		}
		gs := []string{}
		for g := range genres {
			gs = append(gs, g)
		}
		sort.Strings(gs)
		return nil, Info{a.Engine.Catalog.Meta, a.Engine.Catalog.Profiles(), gs, "2025-11-25", "guided scripted web simulation; not live Alexa+ or LLM inference"}, nil
	})
	mcp.AddTool(a.MCP, tool("create_room", "Create a movie-night room. Its random ID is a local collaboration capability; share only with invited members.", false), func(ctx context.Context, req *mcp.CallToolRequest, in CreateInput) (*mcp.CallToolResult, room.Room, error) {
		r, err := a.Store.Create(ctx, in.Name)
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("get_room", "Read the current room revision, member preferences and saved choice receipts.", true), func(ctx context.Context, req *mcp.CallToolRequest, in RoomInput) (*mcp.CallToolResult, room.Room, error) {
		r, err := a.Store.Get(ctx, in.RoomID)
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("set_member", "Add (empty member_id) or edit a member. Every member has likes, explicit genre vetoes and an optional anonymous MovieLens history. Requires current revision and operation key.", false), func(ctx context.Context, req *mcp.CallToolRequest, in MemberInput) (*mcp.CallToolResult, room.Room, error) {
		member := room.Member{ID: in.MemberID, Name: strings.TrimSpace(in.Name), Profile: in.Profile, Likes: in.Likes, Vetoes: in.Vetoes}
		if err := a.Engine.Validate(member); err != nil {
			return nil, room.Room{}, err
		}
		r, err := a.Store.Mutate(ctx, in.RoomID, in.Revision, in.OperationKey, in, func(r *room.Room) error {
			if member.ID == "" {
				if len(r.Members) >= 8 {
					return fmt.Errorf("maximum 8 members")
				}
				member.ID = room.ID()
				r.Members = append(r.Members, member)
				return nil
			}
			for i, m := range r.Members {
				if m.ID == member.ID {
					r.Members[i] = member
					return nil
				}
			}
			return fmt.Errorf("member not found")
		})
		if err == nil {
			a.hub.publish(r.ID, r.Revision)
		}
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("remove_member", "Remove a person from a room using current revision and an idempotency key.", false), func(ctx context.Context, req *mcp.CallToolRequest, in RemoveInput) (*mcp.CallToolResult, room.Room, error) {
		r, err := a.Store.Mutate(ctx, in.RoomID, in.Revision, in.OperationKey, in, func(r *room.Room) error {
			for i, m := range r.Members {
				if m.ID == in.MemberID {
					r.Members = append(r.Members[:i], r.Members[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("member not found")
		})
		if err == nil {
			a.hub.publish(r.ID, r.Revision)
		}
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("set_policy", "Choose whether to exclude any film previously rated by a room member. Vetoes are always honored.", false), func(ctx context.Context, req *mcp.CallToolRequest, in PolicyInput) (*mcp.CallToolResult, room.Room, error) {
		r, err := a.Store.Mutate(ctx, in.RoomID, in.Revision, in.OperationKey, in, func(r *room.Room) error { r.ExcludeSeen = in.ExcludeSeen; return nil })
		if err == nil {
			a.hub.publish(r.ID, r.Revision)
		}
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("shortlist", "Rank eligible films for all members. fair uses 65% minimum estimated enjoyment + 35% average. average and popularity are comparison baselines. Empty results never relax vetoes.", true), func(ctx context.Context, req *mcp.CallToolRequest, in ShortlistInput) (*mcp.CallToolResult, recommend.Result, error) {
		if in.Mode != "fair" && in.Mode != "average" && in.Mode != "popularity" {
			return nil, recommend.Result{}, fmt.Errorf("mode must be fair, average or popularity")
		}
		r, err := a.Store.Get(ctx, in.RoomID)
		if err != nil {
			return nil, recommend.Result{}, err
		}
		limit := in.Limit
		if limit > 30 {
			limit = 30
		}
		return nil, a.Engine.Shortlist(r, limit, in.Mode), nil
	})
	mcp.AddTool(a.MCP, tool("save_choice", "Save the group's explicitly chosen movie with a durable explanation and source revision. Rechecks current constraints; stale choices fail.", false), func(ctx context.Context, req *mcp.CallToolRequest, in SaveInput) (*mcp.CallToolResult, room.Room, error) {
		r, err := a.Store.Mutate(ctx, in.RoomID, in.Revision, in.OperationKey, in, func(r *room.Room) error {
			// Eligibility is independent of which comparison strategy displayed
			// the film. Recheck every hard constraint, not membership in fair top-30.
			result := a.Engine.Shortlist(*r, len(a.Engine.Catalog.IDs), "fair")
			for _, c := range result.Candidates {
				if c.Movie.ID == in.MovieID {
					if len(r.Choices) >= 100 {
						return fmt.Errorf("maximum 100 receipts")
					}
					r.Choices = append(r.Choices, room.Choice{MovieID: c.Movie.ID, Title: c.Movie.Title, FromRevision: r.Revision, SavedAt: time.Now().UTC().Format(time.RFC3339Nano), Explanation: c.Explanation})
					return nil
				}
			}
			return fmt.Errorf("movie does not satisfy current room constraints; refresh before choosing")
		})
		if err == nil {
			a.hub.publish(r.ID, r.Revision)
		}
		return nil, r, err
	})
	mcp.AddTool(a.MCP, tool("search_movies", "Search the local movie catalog by title. No playback links or streaming availability are provided.", true), func(ctx context.Context, req *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
		limit := in.Limit
		if limit < 1 {
			limit = 10
		}
		if limit > 30 {
			limit = 30
		}
		out := SearchOutput{Movies: []catalog.Movie{}}
		q := strings.ToLower(strings.TrimSpace(in.Query))
		for _, id := range a.Engine.Catalog.IDs {
			m := a.Engine.Catalog.Movies[id]
			if strings.Contains(strings.ToLower(m.Title), q) {
				out.Movies = append(out.Movies, m)
				if len(out.Movies) >= limit {
					break
				}
			}
		}
		return nil, out, nil
	})
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return a.MCP }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 64 << 10}))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "data": a.Engine.Catalog.Meta, "protocol": "2025-11-25", "mode": "Alexa+ web simulation"})
	})
	mux.HandleFunc("GET /events/{id}", a.events)
	root, _ := fs.Sub(assets, "web")
	mux.Handle("/", http.FileServer(http.FS(root)))
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("Cache-Control", "no-store")
		host := r.Host
		if !strings.HasPrefix(host, "127.0.0.1:") && !strings.HasPrefix(host, "localhost:") && !strings.HasPrefix(host, "[::1]:") {
			http.Error(w, "localhost only", http.StatusForbidden)
			return
		}
		protected.ServeHTTP(w, r)
	})
}
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.Store.Get(r.Context(), id); err != nil {
		http.Error(w, "room not found", 404)
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", 500)
		return
	}
	ch := make(chan int, 1)
	a.hub.mu.Lock()
	if a.hub.subscribers[id] == nil {
		a.hub.subscribers[id] = map[chan int]bool{}
	}
	if len(a.hub.subscribers[id]) >= 16 {
		a.hub.mu.Unlock()
		http.Error(w, "too many listeners", 429)
		return
	}
	a.hub.subscribers[id][ch] = true
	a.hub.mu.Unlock()
	defer func() {
		a.hub.mu.Lock()
		delete(a.hub.subscribers[id], ch)
		if len(a.hub.subscribers[id]) == 0 {
			delete(a.hub.subscribers, id)
		}
		a.hub.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, ": connected\n\n")
	f.Flush()
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case rev := <-ch:
			if _, err := fmt.Fprintf(w, "data: %d\n\n", rev); err != nil {
				return
			}
			f.Flush()
		case <-tick.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			f.Flush()
		}
	}
}
