package workflow

import (
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"strings"
)

type MCPEmpty struct{}
type MCPMember struct {
	MemberID string          `json:"member_id"`
	Expected int             `json:"expected_room_revision"`
	Key      string          `json:"operation_key"`
	Patch    PreferencePatch `json:"patch"`
}
type MCPPolicy struct {
	Expected int    `json:"expected_room_revision"`
	Key      string `json:"operation_key"`
	Policy   Policy `json:"policy"`
}
type MCPQuery struct {
	Query string `json:"query"`
}
type MCPShortlist struct {
	Mode string `json:"mode,omitempty"`
}
type MCPFilm struct {
	FilmID string `json:"film_id"`
}

func toolResult(v any, e error) (*mcp.CallToolResult, map[string]any, error) {
	if e != nil {
		var body any = &fault{Code: "INTERNAL_ERROR", Message: "tool could not be completed"}
		if f, ok := e.(*fault); ok {
			body = f
		}
		b, _ := json.Marshal(body)
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
	}
	b, _ := json.Marshal(v)
	var out map[string]any
	json.Unmarshal(b, &out)
	return nil, out, nil
}
func (a *App) mcpServer(id, secret string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "CommonReel", Version: "0.2.0"}, &mcp.ServerOptions{SupportedProtocolVersions: []string{"2025-11-25"}, Instructions: "Group-recorded choices may be proxy decisions. Estimates are not consent. Never silently relax conditions. No acceptance, save, deletion, or profile tools. Get the current room revision before explicit updates. Do not guess member identity."})
	t := func(name, description string, read bool) *mcp.Tool {
		closed := false
		return &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: read, DestructiveHint: &closed, OpenWorldHint: &closed}}
	}
	mcp.AddTool(s, t("catalog_info", "Bounded catalog size, genres and source notices.", true), func(ctx context.Context, r *mcp.CallToolRequest, in MCPEmpty) (*mcp.CallToolResult, map[string]any, error) {
		if _, e := a.authorized(ctx, a.db, id, secret); e != nil {
			return toolResult(nil, e)
		}
		genres := []string{}
		for _, f := range a.cfg.Films {
			genres = unique(append(genres, f.Genres...))
		}
		return toolResult(map[string]any{"catalog_total": len(a.cfg.Films), "genres": genres, "region": "US", "notice": "Only verified subscription supply qualifies; individual account entitlements remain unknown."}, nil)
	})
	mcp.AddTool(s, t("get_room", "Current room inputs and revisions, without full history or secrets.", true), func(ctx context.Context, r *mcp.CallToolRequest, in MCPEmpty) (*mcp.CallToolResult, map[string]any, error) {
		v, e := a.current(ctx, id, secret)
		if e != nil {
			return toolResult(nil, e)
		}
		v = sessionView(v)
		for i := range v.Members {
			v.Members[i].SourceProfile = ""
			v.Members[i].Preferences.Ratings = nil
			v.Members[i].Preferences.Likes = nil
		}
		return toolResult(v, nil)
	})
	mcp.AddTool(s, t("set_member", "Patch an explicit existing member's tonight inputs. Cannot add members, record responses or modify profiles.", false), func(ctx context.Context, r *mcp.CallToolRequest, in MCPMember) (*mcp.CallToolResult, map[string]any, error) {
		v, e := a.command(ctx, id, secret, Command{Type: "update_member", Expected: in.Expected, Key: in.Key, MemberID: in.MemberID, Patch: &in.Patch})
		return toolResult(v, e)
	})
	mcp.AddTool(s, t("set_policy", "Set explicitly requested shared runtime, platform IDs and rewatch policy. Read current revision first.", false), func(ctx context.Context, r *mcp.CallToolRequest, in MCPPolicy) (*mcp.CallToolResult, map[string]any, error) {
		v, e := a.command(ctx, id, secret, Command{Type: "update_policy", Expected: in.Expected, Key: in.Key, Policy: &in.Policy})
		return toolResult(v, e)
	})
	mcp.AddTool(s, t("shortlist", "Rank at most six eligible movies. fair, average, catalog_prior; empty results never relax constraints.", true), func(ctx context.Context, r *mcp.CallToolRequest, in MCPShortlist) (*mcp.CallToolResult, map[string]any, error) {
		v, e := a.shortlist(ctx, id, secret, in.Mode)
		return toolResult(v, e)
	})
	mcp.AddTool(s, t("search_movies", "Search only the supported movie catalog; matching a title alone does not prove eligibility.", true), func(ctx context.Context, r *mcp.CallToolRequest, in MCPQuery) (*mcp.CallToolResult, map[string]any, error) {
		if _, e := a.authorized(ctx, a.db, id, secret); e != nil {
			return toolResult(nil, e)
		}
		films := []Film{}
		for _, f := range a.cfg.Films {
			if strings.Contains(strings.ToLower(f.Title), strings.ToLower(in.Query)) {
				films = append(films, f)
			}
			if len(films) == 30 {
				break
			}
		}
		return toolResult(map[string]any{"films": films, "catalog_total": len(a.cfg.Films)}, nil)
	})
	mcp.AddTool(s, t("explain_candidate", "Return checked metadata, supply and estimated score for an eligible film. Does not propose or save it.", true), func(ctx context.Context, r *mcp.CallToolRequest, in MCPFilm) (*mcp.CallToolResult, map[string]any, error) {
		room, e := a.current(ctx, id, secret)
		if e != nil {
			return toolResult(nil, e)
		}
		f, ok := a.film(in.FilmID)
		if !ok {
			return toolResult(nil, fail("NOT_FOUND", 404, "film outside supported catalog"))
		}
		av := a.availability(ctx, f, false)
		if reason := a.eligibility(room, f, av); reason != "" {
			return toolResult(nil, fail("SELECTION_STALE", 409, "film is not eligible: "+reason))
		}
		return toolResult(map[string]any{"candidate": a.scoreCandidate(room, f, av, "fair"), "decision_version": room.Decision, "notice": "Scores are estimates; subscription listing does not verify personal account rights."}, nil)
	})
	return s
}
func (a *App) mcpHandler() http.Handler {
	stream := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return a.mcpServer(r.Header.Get("X-CommonReel-Session"), token(r)) }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 64 << 10})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, e := a.authorized(r.Context(), a.db, r.Header.Get("X-CommonReel-Session"), token(r)); e != nil {
			writeError(w, e)
			return
		}
		stream.ServeHTTP(w, r)
	})
}
