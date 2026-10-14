package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type modelFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}
type modelCall struct {
	Function modelFunction `json:"function"`
}
type modelMessage struct {
	Role      string      `json:"role"`
	Content   string      `json:"content"`
	ToolCalls []modelCall `json:"tool_calls,omitempty"`
	ToolName  string      `json:"tool_name,omitempty"`
}
type AgentInput struct {
	Key     string `json:"operation_key"`
	Message string `json:"message"`
	Speaker string `json:"speaker_member_id,omitempty"`
}
type AgentStep struct {
	Fields   []string `json:"fields,omitempty"`
	Origin   string   `json:"origin,omitempty"`
	Target   string   `json:"target_member_id,omitempty"`
	Tool     string   `json:"tool"`
	OK       bool     `json:"ok"`
	Code     string   `json:"code,omitempty"`
	Revision int      `json:"room_revision,omitempty"`
}
type AgentOutput struct {
	Status  string      `json:"status"`
	Reply   string      `json:"reply"`
	Steps   []AgentStep `json:"steps"`
	Elapsed float64     `json:"elapsed_seconds"`
	Model   string      `json:"model"`
}
type capabilityTransport struct{ secret, id string }

func (t capabilityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.Header = r.Header.Clone()
	copy.Header.Set("Authorization", "Bearer "+t.secret)
	copy.Header.Set("X-CommonReel-Session", t.id)
	return http.DefaultTransport.RoundTrip(copy)
}
func (a *App) SetMCPEndpoint(endpoint string) error {
	u, e := url.Parse(endpoint)
	if e != nil {
		return e
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) {
		return fmt.Errorf("MCP endpoint must be this application's explicit local/private HTTP address")
	}
	a.agentMu.Lock()
	a.mcpEndpoint = endpoint
	a.agentMu.Unlock()
	return nil
}
func (a *App) cancelAgent(id string) {
	a.agentMu.Lock()
	defer a.agentMu.Unlock()
	if cancel := a.agents[id]; cancel != nil {
		cancel()
	}
	delete(a.conversations, id)
}
func (a *App) model(ctx context.Context, messages []modelMessage, tools []any) (modelMessage, error) {
	endpoint := a.cfg.OllamaEndpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	u, e := url.Parse(endpoint)
	if e != nil {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "invalid local model address")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || ip == nil || !ip.IsLoopback() {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "only a loopback Ollama endpoint is supported")
	}
	model := a.cfg.OllamaModel
	if model == "" {
		model = "qwen3.5:4b"
	}
	if strings.Contains(strings.ToLower(model), "cloud") {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "cloud model tags are disabled; choose a downloaded local model")
	}
	b, _ := json.Marshal(map[string]any{"model": model, "messages": messages, "tools": tools, "stream": false, "think": false, "options": map[string]any{"num_ctx": 4096, "num_predict": 768}})
	req, _ := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(endpoint, "/")+"/api/chat", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "local model unavailable or timed out; use the forms")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "local model rejected the request; use the forms")
	}
	var out struct {
		Message modelMessage `json:"message"`
		Error   string       `json:"error"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out); e != nil || out.Error != "" {
		return modelMessage{}, fail("MODEL_UNAVAILABLE", 503, "invalid model response; use the forms")
	}
	return out.Message, nil
}
func (a *App) agentTurn(w http.ResponseWriter, r *http.Request) {
	var in AgentInput
	if e := decode(w, r, &in); e != nil {
		writeError(w, e)
		return
	}
	if e := validKey(in.Key); e != nil {
		writeError(w, e)
		return
	}
	if len(in.Message) < 1 || len(in.Message) > 8192 {
		writeError(w, fail("INVALID_ARGUMENT", 400, "message must have 1-8192 bytes"))
		return
	}
	id, secret := r.PathValue("id"), token(r)
	s, e := a.authorized(r.Context(), a.db, id, secret)
	if e != nil {
		writeError(w, e)
		return
	}
	allowed := map[string]bool{}
	if in.Speaker != "" {
		if memberIndex(s, in.Speaker) < 0 {
			writeError(w, fail("INVALID_ARGUMENT", 400, "speaker is not a current member"))
			return
		}
		allowed[in.Speaker] = true
	}
	for _, m := range s.Members {
		count := 0
		for _, other := range s.Members {
			if strings.EqualFold(other.Name, m.Name) {
				count++
			}
		}
		if strings.Contains(in.Message, m.ID) || (count == 1 && strings.Contains(strings.ToLower(in.Message), strings.ToLower(m.Name))) {
			allowed[m.ID] = true
		}
	}
	lower := strings.ToLower(in.Message)
	explicit := map[string]bool{}
	for _, m := range s.Members {
		named := strings.Contains(strings.ToLower(in.Message), strings.ToLower(m.Name))
		matches := 0
		for _, other := range s.Members {
			if strings.EqualFold(other.Name, m.Name) {
				matches++
			}
		}
		idPattern := regexp.MustCompile(`(^|[^a-zA-Z0-9_])` + regexp.QuoteMeta(m.ID) + `($|[^a-zA-Z0-9_])`)
		if idPattern.MatchString(in.Message) || (named && matches == 1) {
			explicit[m.ID] = true
		}
	}
	if len(explicit) > 0 {
		allowed = explicit
	}
	ambiguous := len(allowed) == 0 && (strings.Contains(lower, "i ") || strings.Contains(lower, "my ") || strings.Contains(in.Message, "我") || strings.Contains(in.Message, "他") || strings.Contains(in.Message, "她"))
	if ambiguous {
		write(w, 200, AgentOutput{Status: "clarification", Reply: "请先选择发言成员，或明确写出成员昵称/ID；我还没有修改任何条件。", Steps: []AgentStep{}})
		return
	}
	a.agentMu.Lock()
	if a.agents[id] != nil {
		a.agentMu.Unlock()
		writeError(w, fail("REVISION_CONFLICT", 409, "one agent turn is already running for this session"))
		return
	}
	turnBudget := 90 * time.Second
	if left := s.ExpiresAt.Sub(a.cfg.Now()); left < turnBudget {
		turnBudget = left
	}
	ctx, cancel := context.WithTimeout(r.Context(), turnBudget)
	a.agents[id] = cancel
	endpoint := a.mcpEndpoint
	previous := append([]modelMessage{}, a.conversations[id]...)
	a.agentMu.Unlock()
	defer func() { cancel(); a.agentMu.Lock(); delete(a.agents, id); a.agentMu.Unlock() }()
	// Turn retries have a minimal durable receipt; raw conversation is never persisted.
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		writeError(w, e)
		return
	}
	rec, ok, e := replay(ctx, tx, id, "agent-"+in.Key, hashRequest(in))
	tx.Rollback()
	if e != nil {
		writeError(w, e)
		return
	}
	if ok {
		write(w, 200, map[string]any{"status": "already_processed", "receipt": rec, "reply": "This turn was already processed. Read the current room state."})
		return
	}
	start := time.Now()
	out := AgentOutput{Status: "model_unavailable", Model: a.cfg.OllamaModel, Steps: []AgentStep{}}
	if endpoint == "" {
		out.Reply = "MCP connection is not configured; the forms remain available."
		write(w, 200, out)
		return
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "CommonReel local agent", Version: "0.2.0"}, nil)
	session, e := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint, DisableStandaloneSSE: true, HTTPClient: &http.Client{Transport: capabilityTransport{secret: secret, id: id}}}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if e != nil {
		out.Reply = "The local tool connection is unavailable; use the forms."
		write(w, 200, out)
		return
	}
	defer session.Close()
	listed, e := session.ListTools(ctx, nil)
	if e != nil {
		out.Reply = "Tools could not be loaded; use the forms."
		write(w, 200, out)
		return
	}
	tools := []any{}
	whitelist := map[string]bool{}
	for _, t := range listed.Tools {
		whitelist[t.Name] = true
		tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": t.InputSchema}})
	}
	view := sessionView(s)
	for i := range view.Members {
		view.Members[i].Preferences.Ratings = nil
		view.Members[i].Preferences.Likes = nil
		view.Members[i].SourceProfile = ""
	}
	roomJSON, _ := json.Marshal(view)
	allowedJSON, _ := json.Marshal(allowed)
	system := "You help a small group choose a movie. Reply in the user's language. Use actual tools before claiming changes or supply facts. Never save, vote, delete, add members, or modify profiles. Never relax constraints without an explicit request. Explicit member IDs allowed for this turn: " + string(allowedJSON) + ". If target is unclear, ask. Get current room revision before writes. Preserve other policy fields. Estimates are not personal consent. Current room (history omitted): " + string(roomJSON)
	messages := []modelMessage{{Role: "system", Content: system}}
	messages = append(messages, previous...)
	messages = append(messages, modelMessage{Role: "user", Content: in.Message})
	calls := 0
	var lastList *Shortlist
	var explanation *Candidate
	explanationDecision := 0
	needsExplain := regexp.MustCompile(`(?i)explain|why|解释|为什么`).MatchString(in.Message)
	for {
		reply, err := a.model(ctx, messages, tools)
		if err != nil {
			out.Status = "model_unavailable"
			out.Reply = err.Error()
			break
		}
		reply.Role = "assistant"
		messages = append(messages, reply)
		if len(reply.ToolCalls) == 0 {
			out.Status = "completed"
			out.Reply = reply.Content
			if needsExplain && explanation == nil && lastList != nil && len(lastList.Candidates) > 0 {
				if calls >= 6 {
					out.Status = "tool_limit"
					out.Reply = "The shortlist is available, but the explanation could not be grounded within six tool calls."
					break
				}
				calls++
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "explain_candidate", Arguments: map[string]any{"film_id": lastList.Candidates[0].Film.ID}})
				step := AgentStep{Tool: "explain_candidate", Origin: "grounding_guard", OK: err == nil && result != nil && !result.IsError}
				out.Steps = append(out.Steps, step)
				if !step.OK {
					out.Status = "tool_rejected"
					out.Reply = "Current explanation facts could not be verified. Refresh the room or use the forms."
					break
				}
				b, _ := json.Marshal(result.StructuredContent)
				var parsed struct {
					Candidate Candidate `json:"candidate"`
					Decision  int       `json:"decision_version"`
				}
				json.Unmarshal(b, &parsed)
				explanation = &parsed.Candidate
				explanationDecision = parsed.Decision
			}
			if lastList != nil || explanation != nil {
				current, e := a.current(ctx, id, secret)
				if e != nil {
					out.Status = "resource_gone"
					out.Reply = "Room is no longer available."
					break
				}
				if lastList != nil && lastList.Decision != current.Decision || explanation != nil && explanationDecision != current.Decision {
					out.Status = "revision_conflict"
					out.Reply = "The room inputs changed during this answer. Refresh and ask for a new shortlist."
					break
				}
				out.Reply = groundedReply(lastList, explanation, current.Policy, regexp.MustCompile(`[\x{4e00}-\x{9fff}]`).MatchString(in.Message))
				break
			}
			if regexp.MustCompile(`(?i)\b(saved|deleted|voted|accepted|recorded)\b|已保存|已删除|已接受|已表态`).MatchString(out.Reply) {
				out.Status = "unverified_reply"
				out.Reply = "保存、表态和删除需要页面按钮。本轮代理未执行这些操作，请查看当前房间记录。"
			}
			wrote := false
			for _, step := range out.Steps {
				if step.OK && (step.Tool == "set_member" || step.Tool == "set_policy") {
					wrote = true
				}
			}
			if !wrote && regexp.MustCompile(`(?i)\b(updated|changed|set successfully)\b|已修改|已更新|已更改`).MatchString(out.Reply) {
				out.Status = "unverified_reply"
				out.Reply = "本轮没有提交任何条件修改，请以页面当前状态为准。"
			}
			if wrote {
				out.Reply = writeReply(out.Steps, regexp.MustCompile(`[\x{4e00}-\x{9fff}]`).MatchString(in.Message))
			} else if regexp.MustCompile(`(?i)recommend|suggest|movie|film|runtime|stream|platform|minutes|推荐|电影|片长|平台|分钟`).MatchString(in.Message + " " + out.Reply) {
				out.Status = "unverified_reply"
				out.Reply = "No verified candidate facts were retrieved in this turn. Please use Find films or ask for a tool-based recommendation; no film or availability claim was accepted."
			}
			break
		}
		rejected := false
		for _, call := range reply.ToolCalls {
			calls++
			fn := call.Function
			if calls > 6 {
				out.Status = "tool_limit"
				out.Reply = "已达到本轮 6 次工具调用上限。请查看实际执行记录或使用表单。"
				rejected = true
				break
			}
			if !whitelist[fn.Name] {
				out.Status = "tool_rejected"
				out.Reply = "模型请求了未授权工具，未执行该操作。"
				out.Steps = append(out.Steps, AgentStep{Tool: fn.Name, Code: "FORBIDDEN"})
				rejected = true
				break
			}
			if fn.Name == "set_member" {
				mid, _ := fn.Arguments["member_id"].(string)
				if !allowed[mid] {
					out.Status = "clarification"
					out.Reply = "这次请求没有明确授权修改该成员，请指定成员后再试。"
					rejected = true
					break
				}
			}
			if _, e := a.authorized(ctx, a.db, id, secret); e != nil {
				out.Status = "resource_gone"
				out.Reply = "场次已结束或资料已撤回。"
				rejected = true
				break
			}
			if fn.Name == "set_member" || fn.Name == "set_policy" {
				current, err := a.current(ctx, id, secret)
				if err == nil {
					err = a.guardAgentWrite(current, fn.Name, fn.Arguments, in.Message)
				}
				if err != nil {
					out.Status = "clarification"
					out.Reply = "The proposed hard-input change was not explicit in your request. No such change was submitted. Please specify the field and value."
					out.Steps = append(out.Steps, AgentStep{Tool: fn.Name, Code: "FORBIDDEN", Origin: "model"})
					rejected = true
					break
				}
				fn.Arguments["operation_key"] = "agent-tool-" + digest(in.Key)[:20] + fmt.Sprint(calls)
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: fn.Name, Arguments: fn.Arguments})
			step := AgentStep{Tool: fn.Name, Origin: "model"}
			if fn.Name == "set_policy" {
				step.Fields = []string{"max_runtime_minutes", "provider_ids", "allow_rewatch"}
			}
			if fn.Name == "set_member" {
				step.Target, _ = fn.Arguments["member_id"].(string)
				if patch, ok := fn.Arguments["patch"].(map[string]any); ok {
					for field := range patch {
						step.Fields = append(step.Fields, field)
					}
				}
			}
			if err != nil {
				step.Code = "PROTOCOL_ERROR"
				out.Status = "tool_rejected"
				out.Reply = "工具调用失败，请刷新后使用表单。"
				out.Steps = append(out.Steps, step)
				rejected = true
				break
			}
			step.OK = !result.IsError
			payload, _ := json.Marshal(result.StructuredContent)
			if result.IsError {
				payload, _ = json.Marshal(result.Content)
				step.Code = "BUSINESS_REJECTION"
			}
			var receipt Receipt
			json.Unmarshal(payload, &receipt)
			step.Revision = receipt.Revision
			if receipt.Code != "" {
				step.Code = receipt.Code
			}
			if !result.IsError && fn.Name == "shortlist" {
				var v Shortlist
				if json.Unmarshal(payload, &v) == nil {
					lastList = &v
				}
			}
			if !result.IsError && fn.Name == "explain_candidate" {
				var v struct {
					Candidate Candidate `json:"candidate"`
					Decision  int       `json:"decision_version"`
				}
				if json.Unmarshal(payload, &v) == nil {
					explanation = &v.Candidate
					explanationDecision = v.Decision
				}
			}
			out.Steps = append(out.Steps, step)
			messages = append(messages, modelMessage{Role: "tool", ToolName: fn.Name, Content: string(payload)})
			if result.IsError {
				out.Status = "tool_rejected"
				out.Reply = "工具拒绝了操作；没有将这次失败记为成功。请刷新条件后重试。"
				rejected = true
				break
			}
		}
		if rejected {
			break
		}
	}
	out.Elapsed = time.Since(start).Seconds()
	if _, e = a.authorized(context.WithoutCancel(r.Context()), a.db, id, secret); e != nil {
		out.Status = "resource_gone"
		out.Reply = "场次已结束，当前对话已清理。"
		out.Steps = []AgentStep{}
	} else {
		a.agentMu.Lock()
		if ctx.Err() == nil {
			previous = append(previous, modelMessage{Role: "user", Content: in.Message}, modelMessage{Role: "assistant", Content: out.Reply})
			if len(previous) > 4 {
				previous = previous[len(previous)-4:]
			}
			a.conversations[id] = previous
		}
		a.agentMu.Unlock()
		a.mu.Lock()
		tx, err := a.db.BeginTx(r.Context(), nil)
		if err == nil {
			if _, err = a.authorized(r.Context(), tx, id, secret); err == nil {
				err = saveReceipt(r.Context(), tx, id, "agent-"+in.Key, hashRequest(in), Receipt{ResourceID: id, Code: out.Status})
			}
			if err == nil {
				tx.Commit()
			} else {
				tx.Rollback()
			}
		}
		a.mu.Unlock()
	}
	if ctx.Err() == context.Canceled {
		out.Status = "cancelled"
		out.Reply = "Room data was withdrawn or this turn was cancelled. Read the current room and use the forms."
		out.Steps = []AgentStep{}
	}
	write(w, 200, out)
}
