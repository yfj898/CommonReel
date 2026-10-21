package workflow

import (
	"context"
	"net/http"
	"sort"
)

type RuntimePreviewInput struct {
	MaxRuntime int `json:"max_runtime_minutes"`
}
type RuntimePreview struct {
	Revision        int         `json:"room_revision"`
	Decision        int         `json:"decision_version"`
	CurrentRuntime  int         `json:"current_max_runtime_minutes"`
	ProposedRuntime int         `json:"max_runtime_minutes"`
	CurrentCount    int         `json:"current_feasible_count"`
	ProposedCount   int         `json:"proposed_feasible_count"`
	NewCount        int         `json:"newly_feasible_count"`
	CurrentUnknown  int         `json:"current_unknown_count"`
	ProposedUnknown int         `json:"proposed_unknown_count"`
	Candidates      []Candidate `json:"candidates"`
	Basis           string      `json:"preview_basis"`
	CatalogTotal    int         `json:"catalog_total"`
}

// runtimePreview is a cache-only observation. Unlike current/shortlist it never
// cleans up a candidate, writes a receipt, or starts a supplier request.
func (a *App) runtimePreview(ctx context.Context, id, secret string, target int) (RuntimePreview, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.authorized(ctx, a.db, id, secret)
	if e != nil {
		return RuntimePreview{}, e
	}
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	return a.runtimePreviewLocked(s, target)
}

// Caller holds cacheMu; applying also holds the existing room transaction.
func (a *App) runtimePreviewLocked(s Session, target int) (RuntimePreview, error) {
	if target <= s.Policy.MaxRuntime || target > 600 {
		return RuntimePreview{}, fail("INVALID_ARGUMENT", 400, "preview runtime must exceed the saved limit and be at most 600 minutes")
	}
	p := RuntimePreview{Revision: s.Revision, Decision: s.Decision, CurrentRuntime: s.Policy.MaxRuntime, ProposedRuntime: target, Candidates: []Candidate{}, CatalogTotal: len(a.cfg.Films)}
	hypothetical := s
	hypothetical.Policy.MaxRuntime = target
	type fact struct {
		Film         Film
		Availability Availability
		Valid        bool
	}
	facts := []fact{}
	now := a.cfg.Now()
	for _, f := range a.cfg.Films {
		av := a.cache[f.ID]
		facts = append(facts, fact{f, av, now.Before(av.ExpiresAt)})
		before, after := a.eligibility(s, f, av), a.eligibility(hypothetical, f, av)
		if before == "" {
			p.CurrentCount++
		}
		if before == "availability_unknown" || before == "metadata_unknown" {
			p.CurrentUnknown++
		}
		if after == "availability_unknown" || after == "metadata_unknown" {
			p.ProposedUnknown++
		}
		if after == "" {
			p.ProposedCount++
			if before != "" {
				p.NewCount++
				p.Candidates = append(p.Candidates, a.scoreCandidate(hypothetical, f, av, "fair"))
			}
		}
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Film.ID < facts[j].Film.ID })
	p.Basis = hashRequest(struct {
		ID                          string
		Revision, Decision, Runtime int
		Facts                       []fact
	}{s.ID, s.Revision, s.Decision, target, facts})
	sort.Slice(p.Candidates, func(i, j int) bool {
		if p.Candidates[i].Score == p.Candidates[j].Score {
			return p.Candidates[i].Film.ID < p.Candidates[j].Film.ID
		}
		return p.Candidates[i].Score > p.Candidates[j].Score
	})
	if len(p.Candidates) > 6 {
		p.Candidates = p.Candidates[:6]
	}
	return p, nil
}
func (a *App) runtimePreviewRoute(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sessions/{id}/runtime-preview", func(w http.ResponseWriter, r *http.Request) {
		var in RuntimePreviewInput
		if e := decode(w, r, &in); e != nil {
			writeError(w, e)
			return
		}
		p, e := a.runtimePreview(r.Context(), r.PathValue("id"), token(r), in.MaxRuntime)
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, p)
	})
}
