package workflow

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type PendingTitle struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Reason      string   `json:"reason"`
	Suggestions []string `json:"suggestions,omitempty"`
}
type ImportReport struct {
	Total     int `json:"total_rows"`
	Duplicate int `json:"duplicate"`
	Matched   int `json:"matched_movie"`
	Episode   int `json:"known_episode"`
	Ambiguous int `json:"ambiguous"`
	Unmatched int `json:"unmatched"`
	Invalid   int `json:"invalid"`
	Unique    int `json:"unique_matched_films"`
}
type History struct {
	Seen        []string       `json:"imported_seen"`
	Pending     []PendingTitle `json:"pending"`
	Report      ImportReport   `json:"report"`
	Fingerprint string         `json:"fingerprint,omitempty"`
}
type Profile struct {
	ID          string      `json:"id"`
	Hash        string      `json:"recovery_hash"`
	Revision    int         `json:"profile_revision"`
	Deleted     bool        `json:"deleted"`
	Preferences Preferences `json:"preferences"`
	History     History     `json:"history"`
}
type ProfileCommand struct {
	Type      string           `json:"type"`
	Expected  int              `json:"expected_profile_revision"`
	Key       string           `json:"operation_key"`
	Patch     *PreferencePatch `json:"patch,omitempty"`
	PendingID string           `json:"pending_id,omitempty"`
	FilmID    string           `json:"film_id,omitempty"`
}

func emptyPreferences() Preferences {
	return Preferences{LikedGenres: []string{}, Vetoes: []string{}, Seen: []string{}, Ratings: map[string]float64{}, Likes: map[string]bool{}}
}
func (a *App) authorizeProfile(ctx context.Context, q queryer, id, secret string, deletedOK bool) (Profile, error) {
	var raw string
	var p Profile
	e := q.QueryRowContext(ctx, "SELECT payload FROM v2_profiles WHERE id=?", id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return p, fail("NOT_FOUND", 404, "profile not found")
	}
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal([]byte(raw), &p); e != nil {
		return p, e
	}
	if !validSecret(secret) || digest(secret) != p.Hash {
		return p, fail("UNAUTHORIZED", 401, "invalid profile recovery code")
	}
	if p.Deleted && !deletedOK {
		return p, fail("RESOURCE_GONE", 410, "profile deleted")
	}
	return p, nil
}
func putProfile(ctx context.Context, tx *sql.Tx, p Profile) error {
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO v2_profiles VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", p.ID, string(b))
	return e
}
func (a *App) createProfile(ctx context.Context, secret, key string) (Receipt, error) {
	if !validSecret(secret) {
		return Receipt{}, fail("INVALID_ARGUMENT", 400, "secure recovery token required")
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
	id := "p_" + digest(secret)[:24]
	if existing, err := a.authorizeProfile(ctx, tx, id, secret, true); err == nil {
		if existing.Deleted {
			return Receipt{}, fail("RESOURCE_GONE", 410, "profile deleted")
		}
	} else {
		var f *fault
		if !errors.As(err, &f) || f.Code != "NOT_FOUND" {
			return Receipt{}, err
		}
	}
	h := hashRequest([]string{"profile", digest(secret)})
	if r, ok, e := replay(ctx, tx, id, key, h); ok || e != nil {
		return r, e
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM v2_profiles WHERE id=?", id).Scan(&n); e != nil {
		return Receipt{}, e
	}
	if n > 0 {
		return Receipt{}, fail("REVISION_CONFLICT", 409, "recovery code already used")
	}
	p := Profile{ID: id, Hash: digest(secret), Revision: 1, Preferences: emptyPreferences(), History: History{Seen: []string{}, Pending: []PendingTitle{}}}
	if e = putProfile(ctx, tx, p); e != nil {
		return Receipt{}, e
	}
	r := Receipt{ResourceID: id, ProfileID: id, ProfileRevision: 1, Code: "created"}
	if e = saveReceipt(ctx, tx, id, key, h, r); e != nil {
		return Receipt{}, e
	}
	return r, tx.Commit()
}
func effectiveSeen(m Member) []string {
	return unique(append(append([]string{}, m.Preferences.Seen...), m.History.Seen...))
}
func unique(xs []string) []string {
	r := []string{}
	for _, x := range xs {
		if !contains(r, x) {
			r = append(r, x)
		}
	}
	return r
}
func sessionView(s Session) Session {
	for i := range s.Members {
		m := &s.Members[i]
		m.SeenCount = len(effectiveSeen(*m))
		m.Preferences.Seen = []string{}
		m.History = History{Report: m.History.Report}
	}
	return s
}
func page(r *http.Request) (int, int, error) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}
	if offset < 0 || limit < 1 || limit > 500 {
		return 0, 0, fail("INVALID_ARGUMENT", 400, "offset >=0, limit 1-500 required")
	}
	return offset, limit, nil
}
func historyPage(h History, p Preferences, offset, limit int) map[string]any {
	seen := unique(append(append([]string{}, p.Seen...), h.Seen...))
	end := offset + limit
	if offset > len(seen) {
		offset = len(seen)
	}
	if end > len(seen) {
		end = len(seen)
	}
	pending := h.Pending
	po := offset
	if po > len(pending) {
		po = len(pending)
	}
	pe := po + limit
	if pe > len(pending) {
		pe = len(pending)
	}
	return map[string]any{"seen": seen[offset:end], "seen_total": len(seen), "pending": pending[po:pe], "pending_total": len(pending), "report": h.Report}
}
func (a *App) resolveHistory(h *History, pendingID, filmID string) error {
	canonical, e := a.canonicalFilm(filmID)
	if e != nil {
		return e
	}
	filmID = canonical
	for i, p := range h.Pending {
		if p.ID == pendingID {
			h.Seen = unique(append(h.Seen, filmID))
			h.Pending = append(h.Pending[:i], h.Pending[i+1:]...)
			return nil
		}
	}
	return fail("NOT_FOUND", 404, "pending title not found")
}
func (a *App) profileCommand(ctx context.Context, id, secret string, c ProfileCommand) (Receipt, error) {
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
	p, e := a.authorizeProfile(ctx, tx, id, secret, c.Type == "delete_profile")
	if e != nil {
		return Receipt{}, e
	}
	h := hashRequest(c)
	if r, ok, e := replay(ctx, tx, id, c.Key, h); ok || e != nil {
		return r, e
	}
	if p.Deleted {
		return Receipt{}, fail("RESOURCE_GONE", 410, "profile deleted")
	}
	if p.Revision != c.Expected {
		return Receipt{}, fail("REVISION_CONFLICT", 409, "profile revision changed")
	}
	affected := []string{}
	switch c.Type {
	case "update_preferences":
		if c.Patch == nil {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "patch required")
		}
		if c.Patch.Vetoes != nil {
			return Receipt{}, fail("INVALID_ARGUMENT", 400, "tonight's vetoes belong to a session, not a reusable profile")
		}
		if e = patchPreferences(&p.Preferences, *c.Patch, a); e != nil {
			return Receipt{}, e
		}
	case "resolve_history":
		if e = a.resolveHistory(&p.History, c.PendingID, c.FilmID); e != nil {
			return Receipt{}, e
		}
	case "delete_profile":
		p.Deleted = true
		p.Preferences = emptyPreferences()
		p.History = History{}
		if e = a.withdrawProfile(ctx, tx, id, &affected); e != nil {
			return Receipt{}, e
		}
	default:
		return Receipt{}, fail("INVALID_ARGUMENT", 400, "unknown profile command")
	}
	p.Revision++
	if e = putProfile(ctx, tx, p); e != nil {
		return Receipt{}, e
	}
	r := Receipt{ResourceID: id, ProfileRevision: p.Revision, Code: "updated"}
	if p.Deleted {
		r.Code = "deleted"
	}
	if e = saveReceipt(ctx, tx, id, c.Key, h, r); e != nil {
		return Receipt{}, e
	}
	e = tx.Commit()
	if e == nil && p.Deleted {
		for _, roomID := range unique(affected) {
			a.cancelAgent(roomID)
		}
	}
	return r, e
}
func (a *App) withdrawProfile(ctx context.Context, tx *sql.Tx, id string, affected *[]string) error {
	rows, e := tx.QueryContext(ctx, "SELECT payload FROM v2_sessions")
	if e != nil {
		return e
	}
	sessions := []Session{}
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var st storedSession
		if e = json.Unmarshal([]byte(raw), &st); e != nil {
			rows.Close()
			return e
		}
		st.Session.InviteHash = st.InviteHash
		sessions = append(sessions, st.Session)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return e
	}
	rows.Close()
	for _, s := range sessions {
		changed := false
		for i := range s.Members {
			m := &s.Members[i]
			if s.Status == "active" && m.SourceProfile == id {
				m.Preferences = emptyPreferences()
				m.History = History{}
				m.SourceProfile = ""
				m.SourceRevision = 0
				changed = true
			}
		}
		if changed {
			*affected = append(*affected, s.ID)
			s.Revision++
			s.Decision++
			s.Candidate = nil
			s.Acceptances = map[string]Acceptance{}
			if e = putSession(ctx, tx, s); e != nil {
				return e
			}
		}
	}
	return nil
}
func normalizeTitle(v string) string { return strings.ToLower(strings.Join(strings.Fields(v), " ")) }
func (a *App) parseCSV(data []byte, column string) (History, error) {
	if len(data) > 5<<20 {
		return History{}, fail("IMPORT_TOO_LARGE", 413, "CSV exceeds 5 MiB")
	}
	if !utf8.Valid(data) {
		return History{}, fail("INVALID_ARGUMENT", 400, "CSV must use UTF-8")
	}
	text := strings.TrimPrefix(string(data), "\ufeff")
	r := csv.NewReader(strings.NewReader(text))
	r.FieldsPerRecord = -1
	head, e := r.Read()
	if e != nil {
		return History{}, fail("INVALID_ARGUMENT", 400, "CSV needs a header")
	}
	for _, field := range head {
		if len(field) > 8192 {
			return History{}, fail("IMPORT_TOO_LARGE", 413, "CSV header field exceeds 8 KiB")
		}
	}
	titleCol, dateCol := -1, -1
	for i, v := range head {
		n := normalizeTitle(v)
		if n == "title" && column == "" || column != "" && v == column {
			titleCol = i
		}
		if n == "date" {
			dateCol = i
		}
	}
	if titleCol < 0 {
		return History{}, fail("INVALID_ARGUMENT", 400, "choose title_column from your CSV header")
	}
	h := History{Seen: []string{}, Pending: []PendingTitle{}, Fingerprint: digest(text)}
	seenRows := map[string]bool{}
	normalizedRows := []string{}
	for {
		row, e := r.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return History{}, fail("INVALID_ARGUMENT", 400, "malformed CSV")
		}
		blank := true
		for _, v := range row {
			if strings.TrimSpace(v) != "" {
				blank = false
			}
			if len(v) > 8192 {
				return History{}, fail("IMPORT_TOO_LARGE", 413, "CSV field exceeds 8 KiB")
			}
		}
		if blank {
			continue
		}
		h.Report.Total++
		if h.Report.Total > 50000 {
			return History{}, fail("IMPORT_TOO_LARGE", 413, "CSV exceeds 50000 rows")
		}
		if titleCol >= len(row) || strings.TrimSpace(row[titleCol]) == "" {
			h.Report.Invalid++
			continue
		}
		title := strings.TrimSpace(row[titleCol])
		date := ""
		if dateCol >= 0 && dateCol < len(row) {
			date = strings.TrimSpace(row[dateCol])
		}
		key := normalizeTitle(title) + "\x00" + date
		normalizedRows = append(normalizedRows, key)
		if seenRows[key] {
			h.Report.Duplicate++
			continue
		}
		seenRows[key] = true
		episode := false
		for _, known := range a.cfg.KnownEpisodes {
			if normalizeTitle(known) == normalizeTitle(title) {
				episode = true
				break
			}
		}
		if episode {
			h.Report.Episode++
			continue
		}
		matches := []string{}
		for _, f := range a.cfg.Films {
			names := append([]string{f.Title, fmtTitleYear(f)}, f.Aliases...)
			for _, n := range names {
				if normalizeTitle(n) == normalizeTitle(title) {
					matches = append(matches, f.ID)
					break
				}
			}
		}
		if len(matches) == 1 {
			h.Report.Matched++
			h.Seen = unique(append(h.Seen, matches[0]))
			continue
		}
		reason := "unmatched"
		if len(matches) > 1 {
			reason = "ambiguous"
			h.Report.Ambiguous++
		} else {
			h.Report.Unmatched++
		}
		h.Pending = append(h.Pending, PendingTitle{ID: digest(key)[:24], Title: title, Reason: reason, Suggestions: matches})
	}
	h.Report.Unique = len(h.Seen)
	sort.Strings(normalizedRows)
	h.Fingerprint = hashRequest(struct {
		Rows    []string
		Catalog string
	}{normalizedRows, hashRequest(a.cfg.Films)})
	return h, nil
}
func fmtTitleYear(f Film) string {
	if f.Year > 0 {
		return f.Title + " (" + strconv.Itoa(f.Year) + ")"
	}
	return f.Title
}
func mergeHistory(old, in History, mode string) History {
	if mode == "replace" {
		return in
	}
	if old.Fingerprint == in.Fingerprint {
		return old
	}
	old.Seen = unique(append(old.Seen, in.Seen...))
	for _, p := range in.Pending {
		found := false
		for _, v := range old.Pending {
			if p.ID == v.ID {
				found = true
			}
		}
		if !found {
			old.Pending = append(old.Pending, p)
		}
	}
	old.Report = in.Report
	old.Fingerprint = in.Fingerprint
	return old
}
func (a *App) importHistory(w http.ResponseWriter, r *http.Request, profile bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	mr, e := r.MultipartReader()
	if e != nil {
		writeError(w, fail("INVALID_ARGUMENT", 400, "multipart CSV required"))
		return
	}
	fields := map[string]string{}
	var data []byte
	for {
		part, e := mr.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil {
			writeError(w, fail("IMPORT_TOO_LARGE", 413, "invalid or oversized multipart request"))
			return
		}
		limit := int64(8192)
		if part.FormName() == "file" {
			limit = 5 << 20
		}
		b, e := io.ReadAll(io.LimitReader(part, limit+1))
		part.Close()
		if e != nil || int64(len(b)) > limit {
			writeError(w, fail("IMPORT_TOO_LARGE", 413, "import field exceeds limit"))
			return
		}
		if part.FormName() == "file" {
			data = b
		} else {
			fields[part.FormName()] = string(b)
		}
	}
	if e = validKey(fields["operation_key"]); e != nil {
		writeError(w, e)
		return
	}
	mode := fields["mode"]
	if mode == "" {
		mode = "append"
	}
	if mode != "append" && mode != "replace" {
		writeError(w, fail("INVALID_ARGUMENT", 400, "mode must be append or replace"))
		return
	}
	parsed, e := a.parseCSV(data, fields["title_column"])
	if e != nil {
		writeError(w, e)
		return
	}
	ctx := r.Context()
	id := r.PathValue("id")
	secret := token(r)
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		writeError(w, e)
		return
	}
	defer tx.Rollback()
	var p Profile
	var s Session
	if profile {
		p, e = a.authorizeProfile(ctx, tx, id, secret, false)
	} else {
		s, e = a.authorized(ctx, tx, id, secret)
	}
	if e != nil {
		writeError(w, e)
		return
	}
	hash := hashRequest([]string{"import", digest(string(data)), mode, fields["title_column"], fields["target_member_id"], fields["expected_profile_revision"], fields["expected_room_revision"]})
	if rec, ok, err := replay(ctx, tx, id, fields["operation_key"], hash); ok || err != nil {
		if err != nil {
			writeError(w, err)
		} else {
			write(w, 200, rec)
		}
		return
	}
	rec := Receipt{ResourceID: id, Code: "imported"}
	if profile {
		rev, _ := strconv.Atoi(fields["expected_profile_revision"])
		if p.Revision != rev {
			writeError(w, fail("REVISION_CONFLICT", 409, "profile revision changed"))
			return
		}
		before := hashRequest(p.History)
		p.History = mergeHistory(p.History, parsed, mode)
		if before != hashRequest(p.History) {
			p.Revision++
		} else {
			rec.Code = "no_change"
		}
		rec.ProfileRevision = p.Revision
		e = putProfile(ctx, tx, p)
	} else {
		rev, _ := strconv.Atoi(fields["expected_room_revision"])
		i := memberIndex(s, fields["target_member_id"])
		if rev != s.Revision {
			writeError(w, fail("REVISION_CONFLICT", 409, "room revision changed"))
			return
		}
		if i < 0 {
			writeError(w, fail("NOT_FOUND", 404, "member not found"))
			return
		}
		before := hashRequest(s.Members[i].History)
		s.Members[i].History = mergeHistory(s.Members[i].History, parsed, mode)
		if before != hashRequest(s.Members[i].History) {
			s.Revision++
			s.Decision++
			s.Candidate = nil
			s.Acceptances = map[string]Acceptance{}
		} else {
			rec.Code = "no_change"
		}
		rec.Revision = s.Revision
		rec.Decision = s.Decision
		e = putSession(ctx, tx, s)
	}
	if e == nil {
		e = saveReceipt(ctx, tx, id, fields["operation_key"], hash, rec)
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		writeError(w, e)
		return
	}
	write(w, 200, rec)
}
func (a *App) profileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Recovery string `json:"recovery_token"`
			Key      string `json:"operation_key"`
		}
		if e := decode(w, r, &in); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.createProfile(r.Context(), in.Recovery, in.Key)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 201, v)
	})
	mux.HandleFunc("GET /api/v1/profiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, e := a.authorizeProfile(r.Context(), a.db, r.PathValue("id"), token(r), false)
		if e != nil {
			writeError(w, e)
			return
		}
		offset, limit, e := page(r)
		if e != nil {
			writeError(w, e)
			return
		}
		v := historyPage(p.History, p.Preferences, offset, limit)
		v["id"] = p.ID
		v["profile_revision"] = p.Revision
		v["seen_count"] = v["seen_total"]
		prefs := p.Preferences
		prefs.Seen = []string{}
		v["preferences"] = prefs
		write(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/profiles/{id}/commands", func(w http.ResponseWriter, r *http.Request) {
		var c ProfileCommand
		if e := decode(w, r, &c); e != nil {
			writeError(w, e)
			return
		}
		v, e := a.profileCommand(r.Context(), r.PathValue("id"), token(r), c)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, v)
	})
	mux.HandleFunc("POST /api/v1/profiles/{id}/history-imports", func(w http.ResponseWriter, r *http.Request) { a.importHistory(w, r, true) })
	mux.HandleFunc("POST /api/v1/sessions/{id}/history-imports", func(w http.ResponseWriter, r *http.Request) { a.importHistory(w, r, false) })
	mux.HandleFunc("GET /api/v1/sessions/{id}/members/{member}/history", func(w http.ResponseWriter, r *http.Request) {
		s, e := a.authorized(r.Context(), a.db, r.PathValue("id"), token(r))
		if e != nil {
			writeError(w, e)
			return
		}
		i := memberIndex(s, r.PathValue("member"))
		if i < 0 {
			writeError(w, fail("NOT_FOUND", 404, "member not found"))
			return
		}
		o, l, e := page(r)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, historyPage(s.Members[i].History, s.Members[i].Preferences, o, l))
	})
}
