package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sweep removes expired personal session copies. Access checks stop reads immediately,
// independently of this physical cleanup interval.
func (a *App) Sweep(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "DELETE FROM v2_supply WHERE julianday(json_extract(payload,'$.fetched_at')) <= julianday(?)", a.cfg.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339Nano)); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, "SELECT payload FROM v2_sessions")
	if e != nil {
		return e
	}
	all := []Session{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return e
		}
		var st storedSession
		if e = json.Unmarshal([]byte(raw), &st); e != nil {
			rows.Close()
			return e
		}
		st.Session.InviteHash = st.InviteHash
		all = append(all, st.Session)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, s := range all {
		if s.Status == "active" && !a.cfg.Now().Before(s.ExpiresAt) {
			s.Status = "expired"
			s.Name = ""
			s.Members = []Member{}
			s.Policy = Policy{}
			s.Candidate = nil
			s.Acceptances = map[string]Acceptance{}
			s.Revision++
			s.Decision++
			if e = putSession(ctx, tx, s); e != nil {
				return e
			}
			a.cancelAgent(s.ID)
		}
	}
	return tx.Commit()
}
func (a *App) cleanupLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-a.stop:
			return
		case <-t.C:
			a.Sweep(context.Background())
		}
	}
}
func (a *App) events(w http.ResponseWriter, r *http.Request) {
	id, secret := r.PathValue("id"), token(r)
	if _, e := a.authorized(r.Context(), a.db, id, secret); e != nil {
		writeError(w, e)
		return
	}
	a.streamMu.Lock()
	if a.streams[id] >= 16 {
		a.streamMu.Unlock()
		writeError(w, fail("QUOTA_EXHAUSTED", 429, "at most 16 streams per session"))
		return
	}
	a.streams[id]++
	a.streamMu.Unlock()
	defer func() { a.streamMu.Lock(); a.streams[id]--; a.streamMu.Unlock() }()
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fail("INTERNAL_ERROR", 500, "streaming unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	last := -1
	t := time.NewTicker(time.Second)
	defer t.Stop()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		s, e := a.current(r.Context(), id, secret)
		if e != nil {
			fmt.Fprint(w, "event: ended\ndata: {\"status\":\"closed_or_expired\"}\n\n")
			flusher.Flush()
			return
		}
		if s.Revision != last {
			b, _ := json.Marshal(map[string]any{"room_revision": s.Revision, "decision_version": s.Decision, "status": s.Status})
			fmt.Fprintf(w, "event: revision\ndata: %s\n\n", b)
			flusher.Flush()
			last = s.Revision
		}
		select {
		case <-r.Context().Done():
			return
		case <-a.stop:
			return
		case <-t.C:
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
