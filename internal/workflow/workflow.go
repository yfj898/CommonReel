package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Database       string
	Now            func() time.Time
	AllowedHosts   []string
	Films          []Film
	Supplier       Supplier
	OllamaEndpoint string
	OllamaModel    string
	InviteOrigin   string
	KnownEpisodes  []string
}
type App struct {
	db            *sql.DB
	mu            sync.Mutex
	cfg           Config
	cacheMu       sync.Mutex
	cache         map[string]Availability
	saveMu        sync.Mutex
	streamMu      sync.Mutex
	streams       map[string]int
	stop          chan struct{}
	closeOnce     sync.Once
	agentMu       sync.Mutex
	agents        map[string]context.CancelFunc
	conversations map[string][]modelMessage
	mcpEndpoint   string
	supplySlots   chan struct{}
}
type Member struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Preferences    Preferences `json:"preferences"`
	SourceProfile  string      `json:"source_profile_id,omitempty"`
	SourceRevision int         `json:"source_profile_revision,omitempty"`
	History        History     `json:"history"`
	SeenCount      int         `json:"seen_count"`
}
type Session struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	InviteHash  string                `json:"-"`
	Status      string                `json:"status"`
	Revision    int                   `json:"room_revision"`
	Decision    int                   `json:"decision_version"`
	ExpiresAt   time.Time             `json:"expires_at"`
	Members     []Member              `json:"members"`
	Policy      Policy                `json:"policy"`
	Candidate   *ActiveCandidate      `json:"active_candidate"`
	Acceptances map[string]Acceptance `json:"acceptances"`
	LastChoice  *ChoiceSummary        `json:"last_choice,omitempty"`
}

func (s *Session) advanceDecision(candidate *ActiveCandidate) {
	s.Revision++
	s.Decision++
	s.Candidate = candidate
	s.Acceptances = map[string]Acceptance{}
}

type storedSession struct {
	Session
	InviteHash string `json:"invite_hash"`
}
type Receipt struct {
	SessionID       string     `json:"session_id,omitempty"`
	ProfileID       string     `json:"profile_id,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	ResourceID      string     `json:"resource_id"`
	Revision        int        `json:"room_revision,omitempty"`
	Decision        int        `json:"decision_version,omitempty"`
	ProfileRevision int        `json:"profile_revision,omitempty"`
	Code            string     `json:"code"`
	Replayed        bool       `json:"replayed"`
	SummaryID       string     `json:"summary_id,omitempty"`
}
type Command struct {
	MaxRuntime     int              `json:"max_runtime_minutes,omitempty"`
	PreviewBasis   string           `json:"preview_basis,omitempty"`
	Type           string           `json:"type"`
	Expected       int              `json:"expected_room_revision"`
	Key            string           `json:"operation_key"`
	Name           string           `json:"name,omitempty"`
	MemberID       string           `json:"member_id,omitempty"`
	TargetMemberID string           `json:"target_member_id,omitempty"`
	Actor          string           `json:"actor_member_id,omitempty"`
	Patch          *PreferencePatch `json:"patch,omitempty"`
	Policy         *Policy          `json:"policy,omitempty"`
	FilmID         string           `json:"film_id,omitempty"`
	Decision       int              `json:"decision_version,omitempty"`
	Response       string           `json:"response,omitempty"`
	ProfileID      string           `json:"profile_id,omitempty"`
	Recovery       string           `json:"recovery_token,omitempty"`
	PendingID      string           `json:"pending_id,omitempty"`
}
type fault struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Status    int    `json:"-"`
	Retryable bool   `json:"retryable"`
}

func (f *fault) Error() string { return f.Code + ": " + f.Message }

func fail(code string, status int, msg string) error {
	return &fault{Code: code, Status: status, Message: msg}
}
func digest(v string) string   { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func hashRequest(v any) string { b, _ := json.Marshal(v); return digest(string(b)) }
func token(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}
func validSecret(v string) bool { return len(v) >= 32 && len(v) <= 256 }
func validKey(v string) error {
	if len(v) < 8 || len(v) > 128 {
		return fail("INVALID_ARGUMENT", 400, "operation_key must have 8-128 bytes")
	}
	return nil
}
func Open(cfg Config) (*App, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Database == "" {
		cfg.Database = ":memory:"
	}
	if cfg.OllamaEndpoint == "" {
		cfg.OllamaEndpoint = "http://127.0.0.1:11434"
	}
	if cfg.OllamaModel == "" {
		cfg.OllamaModel = "qwen3.5:4b"
	}
	db, e := sql.Open("sqlite", cfg.Database)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	if _, e = db.Exec(`CREATE TABLE IF NOT EXISTS v2_profiles(id TEXT PRIMARY KEY,payload TEXT NOT NULL)`); e != nil {
		db.Close()
		return nil, e
	}
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000", `CREATE TABLE IF NOT EXISTS v2_sessions(id TEXT PRIMARY KEY,payload TEXT NOT NULL)`, `CREATE TABLE IF NOT EXISTS v2_operations(scope TEXT NOT NULL,key TEXT NOT NULL,hash TEXT NOT NULL,result TEXT NOT NULL,PRIMARY KEY(scope,key))`, `CREATE TABLE IF NOT EXISTS v2_summaries(id TEXT PRIMARY KEY,capability_hash TEXT NOT NULL,payload TEXT NOT NULL)`} {
		if _, e = db.Exec(q); e != nil {
			db.Close()
			return nil, e
		}
	}
	a := &App{db: db, cfg: cfg, cache: map[string]Availability{}, streams: map[string]int{}, stop: make(chan struct{})}
	a.agents = map[string]context.CancelFunc{}
	a.supplySlots = make(chan struct{}, 4)
	a.conversations = map[string][]modelMessage{}
	if e = a.loadSupply(context.Background()); e != nil {
		db.Close()
		return nil, e
	}
	if e = a.Sweep(context.Background()); e != nil {
		db.Close()
		return nil, e
	}
	go a.cleanupLoop()
	return a, nil
}
func (a *App) Close() error {
	var err error
	a.closeOnce.Do(func() { close(a.stop); err = a.db.Close() })
	return err
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, e error) {
	var f *fault
	if errors.As(e, &f) {
		write(w, f.Status, f)
	} else {
		write(w, 500, &fault{Code: "INTERNAL_ERROR", Message: "request could not be completed"})
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return fail("INVALID_ARGUMENT", 400, "invalid JSON or unsupported fields")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return fail("INVALID_ARGUMENT", 400, "exactly one JSON object required")
	}
	return nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readSession(ctx context.Context, q queryer, id string) (Session, error) {
	var raw string
	var s storedSession
	e := q.QueryRowContext(ctx, "SELECT payload FROM v2_sessions WHERE id=?", id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return Session{}, fail("NOT_FOUND", 404, "session not found")
	}
	if e != nil {
		return Session{}, e
	}
	if e = json.Unmarshal([]byte(raw), &s); e != nil {
		return Session{}, e
	}
	s.Session.InviteHash = s.InviteHash
	return s.Session, nil
}
func readSessions(ctx context.Context, tx *sql.Tx) ([]Session, error) {
	rows, e := tx.QueryContext(ctx, "SELECT payload FROM v2_sessions")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	sessions := []Session{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var s storedSession
		if e = json.Unmarshal([]byte(raw), &s); e != nil {
			return nil, e
		}
		s.Session.InviteHash = s.InviteHash
		sessions = append(sessions, s.Session)
	}
	return sessions, rows.Err()
}
func putSession(ctx context.Context, tx *sql.Tx, s Session) error {
	b, e := json.Marshal(storedSession{Session: s, InviteHash: s.InviteHash})
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO v2_sessions(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", s.ID, string(b))
	return e
}
func replay(ctx context.Context, tx *sql.Tx, scope, key, hash string) (Receipt, bool, error) {
	var h, b string
	e := tx.QueryRowContext(ctx, "SELECT hash,result FROM v2_operations WHERE scope=? AND key=?", scope, key).Scan(&h, &b)
	if errors.Is(e, sql.ErrNoRows) {
		return Receipt{}, false, nil
	}
	if e != nil {
		return Receipt{}, false, e
	}
	if h != hash {
		return Receipt{}, false, fail("OPERATION_KEY_REUSED", 409, "operation key belongs to a different request")
	}
	var rec Receipt
	e = json.Unmarshal([]byte(b), &rec)
	rec.Replayed = true
	return rec, true, e
}
func saveReceipt(ctx context.Context, tx *sql.Tx, scope, key, hash string, rec Receipt) error {
	b, _ := json.Marshal(rec)
	_, e := tx.ExecContext(ctx, "INSERT INTO v2_operations VALUES(?,?,?,?)", scope, key, hash, string(b))
	return e
}
func (a *App) create(ctx context.Context, name, secret, key string) (Receipt, error) {
	if !validSecret(secret) || strings.TrimSpace(name) == "" || len(name) > 80 {
		return Receipt{}, fail("INVALID_ARGUMENT", 400, "name and a secure invite token are required")
	}
	if e := validKey(key); e != nil {
		return Receipt{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return Receipt{}, e
	}
	defer tx.Rollback()
	id := "s_" + digest(secret)[:24]
	exists := false
	if existing, err := readSession(ctx, tx, id); err == nil {
		exists = true
		if existing.InviteHash != digest(secret) {
			return Receipt{}, fail("REVISION_CONFLICT", 409, "invite identifier collision; generate another token")
		}
		if existing.Status != "active" || !a.cfg.Now().Before(existing.ExpiresAt) {
			return Receipt{}, fail("RESOURCE_GONE", 410, "session closed or expired")
		}
	} else {
		var f *fault
		if !errors.As(err, &f) || f.Code != "NOT_FOUND" {
			return Receipt{}, err
		}
	}
	h := hashRequest([]string{"create", name, digest(secret)})
	if rec, ok, e := replay(ctx, tx, id, key, h); ok || e != nil {
		return rec, e
	}
	if exists {
		return Receipt{}, fail("REVISION_CONFLICT", 409, "invite token already used")
	}
	s := Session{ID: id, Name: name, InviteHash: digest(secret), Status: "active", Revision: 1, Decision: 1, ExpiresAt: a.cfg.Now().UTC().Add(24 * time.Hour), Members: []Member{}, Acceptances: map[string]Acceptance{}}
	if e = putSession(ctx, tx, s); e != nil {
		return Receipt{}, e
	}
	rec := Receipt{ResourceID: id, SessionID: id, ExpiresAt: &s.ExpiresAt, Revision: 1, Decision: 1, Code: "created"}
	if e = saveReceipt(ctx, tx, id, key, h, rec); e != nil {
		return Receipt{}, e
	}
	return rec, tx.Commit()
}
func (a *App) authorized(ctx context.Context, q queryer, id, secret string) (Session, error) {
	s, e := readSession(ctx, q, id)
	if e != nil {
		return s, e
	}
	if !validSecret(secret) || digest(secret) != s.InviteHash {
		return s, fail("UNAUTHORIZED", 401, "invalid room capability")
	}
	if s.Status != "active" || !a.cfg.Now().Before(s.ExpiresAt) {
		return s, fail("RESOURCE_GONE", 410, "session closed or expired")
	}
	return s, nil
}
func (a *App) command(ctx context.Context, id, secret string, c Command) (Receipt, error) {
	if c.Type != "close_session" {
		if _, e := a.current(ctx, id, secret); e != nil {
			return Receipt{}, e
		}
	}
	var prepared *ActiveCandidate
	if c.Type == "save_choice" {
		a.saveMu.Lock()
		defer a.saveMu.Unlock()
	}
	if c.Type == "save_choice" || c.Type == "propose_candidate" {
		s, e := a.authorized(ctx, a.db, id, secret)
		if e != nil {
			return Receipt{}, e
		}
		tx, e := a.db.BeginTx(ctx, nil)
		if e != nil {
			return Receipt{}, e
		}
		rec, ok, e := replay(ctx, tx, id, c.Key, hashRequest(c))
		tx.Rollback()
		if ok || e != nil {
			return rec, e
		}
		if c.Expected != s.Revision {
			return Receipt{}, fail("REVISION_CONFLICT", 409, "refresh the current room revision")
		}
		if c.Type == "save_choice" {
			if e = readyToSave(s, c); e != nil {
				return Receipt{}, e
			}
			var summaryID string
			if a.db.QueryRowContext(ctx, "SELECT id FROM v2_summaries WHERE capability_hash=? AND json_extract(payload,'$.decision_version')=? AND json_extract(payload,'$.film_id')=? LIMIT 1", s.InviteHash, s.Decision, c.FilmID).Scan(&summaryID) == nil {
				a.mu.Lock()
				tx, e := a.db.BeginTx(ctx, nil)
				if e != nil {
					a.mu.Unlock()
					return Receipt{}, e
				}
				defer tx.Rollback()
				current, e := a.authorized(ctx, tx, id, secret)
				if e == nil && current.Revision != c.Expected {
					e = fail("REVISION_CONFLICT", 409, "room revision changed")
				}
				receipt := Receipt{ResourceID: id, Revision: current.Revision, Decision: current.Decision, SummaryID: summaryID, Code: "already_recorded"}
				if e == nil {
					e = saveReceipt(ctx, tx, id, c.Key, hashRequest(c), receipt)
				}
				if e == nil {
					e = tx.Commit()
				}
				a.mu.Unlock()
				return receipt, e
			}
		}
		f, ok := a.film(c.FilmID)
		if !ok {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "film is outside the supported catalog")
		}
		av := a.availability(ctx, f, c.Type == "save_choice")
		if !av.Known || !a.cfg.Now().Before(av.ExpiresAt) {
			return Receipt{}, fail("AVAILABILITY_UNKNOWN", 503, "current subscription supply is unknown; no choice saved")
		}
		reason := a.eligibility(s, f, av)
		prepared = &ActiveCandidate{Title: f.Title, FilmID: f.ID, Decision: s.Decision + 1, CatalogVersion: f.Version, FactsVersion: av.Version, VerifiedAt: av.FetchedAt, ExpiresAt: av.ExpiresAt}
		if reason != "" || (c.Type == "save_choice" && (s.Candidate.FactsVersion != av.Version || s.Candidate.CatalogVersion != f.Version)) {
			if c.Type == "save_choice" {
				return a.invalidateSelection(ctx, id, secret, c.Expected)
			}
			return Receipt{}, fail("SELECTION_STALE", 409, "film does not satisfy current conditions: "+reason)
		}
	}
	return a.commitCommand(ctx, id, secret, c, prepared)
}
func (a *App) commitCommand(ctx context.Context, id, secret string, c Command, prepared *ActiveCandidate) (Receipt, error) {
	if e := validKey(c.Key); e != nil {
		return Receipt{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return Receipt{}, e
	}
	defer tx.Rollback()
	s, e := a.authorized(ctx, tx, id, secret)
	if e != nil && c.Type == "close_session" {
		var f *fault
		if errors.As(e, &f) && f.Code == "RESOURCE_GONE" {
			old, readErr := readSession(ctx, tx, id)
			if readErr == nil && digest(secret) == old.InviteHash {
				if rec, ok, err := replay(ctx, tx, id, c.Key, hashRequest(c)); ok || err != nil {
					return rec, err
				}
			}
		}
	}
	if e != nil {
		return Receipt{}, e
	}
	h := hashRequest(c)
	if rec, ok, e := replay(ctx, tx, id, c.Key, h); ok || e != nil {
		return rec, e
	}
	if c.Expected != s.Revision {
		return Receipt{}, fail("REVISION_CONFLICT", 409, "refresh the current room revision")
	}
	changed := true
	basis := true
	summaryID := ""
	switch c.Type {
	case "apply_profile":
		i := memberIndex(s, c.MemberID)
		if i < 0 {
			return Receipt{}, fail("NOT_FOUND", 404, "member not found")
		}
		p, e := a.authorizeProfile(ctx, tx, c.ProfileID, c.Recovery, false)
		if e != nil {
			return Receipt{}, e
		}
		s.Members[i].Preferences = p.Preferences
		s.Members[i].History = p.History
		s.Members[i].SourceProfile = p.ID
		s.Members[i].SourceRevision = p.Revision
	case "resolve_history":
		i := memberIndex(s, c.MemberID)
		if i < 0 {
			return Receipt{}, fail("NOT_FOUND", 404, "member not found")
		}
		if e := a.resolveHistory(&s.Members[i].History, c.PendingID, c.FilmID); e != nil {
			return Receipt{}, e
		}
	case "join_member":
		if len(s.Members) >= 4 {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "at most four members")
		}
		if strings.TrimSpace(c.Name) == "" || len(c.Name) > 80 {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "member name required")
		}
		s.Members = append(s.Members, Member{ID: fmt.Sprintf("m%d", s.Revision), Name: c.Name})
	case "apply_runtime_preview":
		a.cacheMu.Lock()
		defer a.cacheMu.Unlock()
		preview, err := a.runtimePreviewLocked(s, c.MaxRuntime)
		if err != nil {
			return Receipt{}, err
		}
		if c.PreviewBasis == "" || c.PreviewBasis != preview.Basis {
			return Receipt{}, fail("PREVIEW_STALE", 409, "preview inputs or cached facts changed; run the preview again")
		}
		if preview.NewCount == 0 {
			return Receipt{}, fail("NO_NEW_CANDIDATES", 409, "no additional verified film qualifies; review the saved limits instead")
		}
		s.Policy.MaxRuntime = c.MaxRuntime
	case "update_policy":
		if c.Policy == nil || c.Policy.MaxRuntime < 1 || c.Policy.MaxRuntime > 600 || len(c.Policy.ProviderIDs) == 0 {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "runtime 1-600 and at least one provider are required")
		}
		for _, p := range c.Policy.ProviderIDs {
			if p < 1 {
				return Receipt{}, fail("INVALID_ARGUMENT", 400, "invalid provider ID")
			}
		}
		changed = hashRequest(s.Policy) != hashRequest(*c.Policy)
		s.Policy = *c.Policy
	case "update_member":
		i := memberIndex(s, c.MemberID)
		if i < 0 || c.Patch == nil {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "explicit existing member and patch required")
		}
		old := hashRequest(s.Members[i])
		if e := patchPreferences(&s.Members[i].Preferences, *c.Patch, a); e != nil {
			return Receipt{}, e
		}
		changed = old != hashRequest(s.Members[i])
	case "remove_member":
		i := memberIndex(s, c.MemberID)
		if i < 0 {
			return Receipt{}, fail("NOT_FOUND", 404, "member not found")
		}
		s.Members = append(s.Members[:i], s.Members[i+1:]...)
	case "propose_candidate":
		if prepared == nil || len(s.Members) < 2 {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "two to four members and an eligible candidate required")
		}
		if s.Candidate != nil && s.Candidate.FilmID == prepared.FilmID && s.Candidate.CatalogVersion == prepared.CatalogVersion && s.Candidate.FactsVersion == prepared.FactsVersion {
			changed = false
			prepared.Decision = s.Decision
		}
		s.Candidate = prepared
	case "record_acceptance":
		basis = false
		if s.Candidate == nil || s.Candidate.FilmID != c.FilmID || s.Decision != c.Decision || !a.cfg.Now().Before(s.Candidate.ExpiresAt) {
			return Receipt{}, fail("SELECTION_STALE", 409, "candidate or version changed")
		}
		if memberIndex(s, c.TargetMemberID) < 0 || (c.Response != "accept" && c.Response != "reject") {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "explicit member and accept/reject required")
		}
		s.Acceptances[c.TargetMemberID] = Acceptance{MemberID: c.TargetMemberID, FilmID: c.FilmID, Decision: c.Decision, Response: c.Response, Actor: c.Actor, At: a.cfg.Now().UTC()}
	case "save_choice":
		basis = false
		if e := readyToSave(s, c); e != nil {
			return Receipt{}, e
		}
		summaryID = "choice_" + digest(id + c.Key)[:24]
		summary := ChoiceSummary{ID: summaryID, FilmID: c.FilmID, At: a.cfg.Now().UTC(), Policy: s.Policy, FactsVersion: prepared.FactsVersion, Decision: s.Decision, Label: "group-recorded choice"}
		s.LastChoice = &summary
		b, _ := json.Marshal(summary)
		if _, e = tx.ExecContext(ctx, "INSERT INTO v2_summaries VALUES(?,?,?)", summaryID, s.InviteHash, string(b)); e != nil {
			return Receipt{}, e
		}
		s.Candidate.VerifiedAt = prepared.VerifiedAt
		s.Candidate.ExpiresAt = prepared.ExpiresAt
	case "close_session":
		s.Status = "closed"
		s.Members = []Member{}
		s.Name = ""
		s.Policy = Policy{}
	default:
		return Receipt{}, fail("INVALID_ARGUMENT", 400, "unknown command")
	}
	code := "updated"
	if changed {
		if basis {
			var next *ActiveCandidate
			if c.Type == "propose_candidate" {
				next = prepared
			}
			s.advanceDecision(next)
		} else {
			s.Revision++
		}
	} else {
		code = "no_change"
	}
	if e = putSession(ctx, tx, s); e != nil {
		return Receipt{}, e
	}
	rec := Receipt{ResourceID: id, Revision: s.Revision, Decision: s.Decision, Code: code, SummaryID: summaryID}
	if e = saveReceipt(ctx, tx, id, c.Key, h, rec); e != nil {
		return Receipt{}, e
	}
	e = tx.Commit()
	if e == nil && (c.Type == "close_session" || c.Type == "remove_member" || c.Type == "apply_profile") {
		a.cancelAgent(id)
	}
	return rec, e
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mcp", a.mcpHandler())
	mux.HandleFunc("POST /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name   string `json:"name"`
			Invite string `json:"invite_token"`
			Key    string `json:"operation_key"`
		}
		if e := decode(w, r, &in); e != nil {
			writeError(w, e)
			return
		}
		rec, e := a.create(r.Context(), in.Name, in.Invite, in.Key)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 201, rec)
	})
	mux.HandleFunc("GET /api/v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, e := a.current(r.Context(), r.PathValue("id"), token(r))
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, sessionView(s))
	})
	mux.HandleFunc("POST /api/v1/sessions/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		var c Command
		if e := decode(w, r, &c); e != nil {
			writeError(w, e)
			return
		}
		rec, e := a.command(r.Context(), r.PathValue("id"), token(r), c)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, rec)
	})
	a.runtimePreviewRoute(mux)
	a.catalogRoutes(mux)
	a.profileRoutes(mux)
	a.webRoutes(mux)
	mux.HandleFunc("GET /api/v1/sessions/{id}/events", a.events)
	mux.HandleFunc("POST /api/v1/sessions/{id}/agent-turns", a.agentTurn)
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, e := net.SplitHostPort(host); e == nil {
			host = h
		}
		allowed := host == "localhost" || host == "127.0.0.1" || host == "::1"
		for _, h := range a.cfg.AllowedHosts {
			if host == h {
				allowed = true
			}
		}
		if !allowed {
			http.Error(w, "host not allowed", 403)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		protected.ServeHTTP(w, r)
	})
}
