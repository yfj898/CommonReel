package workflow

import (
	"context"
)

// Reading current decision state also retires stale evidence. This prevents old
// responses surviving between a supplier failure/expiry and the next save.
func (a *App) candidateState(s Session) (bool, *Availability) {
	if s.Candidate == nil {
		return false, nil
	}
	c := s.Candidate
	f, ok := a.film(c.FilmID)
	if !ok || f.Version != c.CatalogVersion {
		return true, nil
	}
	a.cacheMu.Lock()
	av, cached := a.cache[c.FilmID]
	a.cacheMu.Unlock()
	if cached {
		if !av.Known || !a.cfg.Now().Before(av.ExpiresAt) || av.Version != c.FactsVersion {
			return true, nil
		}
		if av.ExpiresAt.After(c.ExpiresAt) || av.FetchedAt.After(c.VerifiedAt) {
			return false, &av
		}
	}
	if !a.cfg.Now().Before(c.ExpiresAt) {
		return true, nil
	}
	return false, nil
}
func (a *App) current(ctx context.Context, id, secret string) (Session, error) {
	s, e := a.authorized(ctx, a.db, id, secret)
	if e != nil {
		return s, e
	}
	if s.Candidate != nil && s.Candidate.Title == "" {
		if f, ok := a.film(s.Candidate.FilmID); ok {
			s.Candidate.Title = f.Title
		}
	}
	bad, renew := a.candidateState(s)
	if !bad && renew == nil {
		return s, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return s, e
	}
	defer tx.Rollback()
	s, e = a.authorized(ctx, tx, id, secret)
	if e != nil {
		return s, e
	}
	bad, renew = a.candidateState(s)
	if bad {
		s.advanceDecision(nil)
	} else if renew != nil {
		s.Revision++
		s.Candidate.ExpiresAt = renew.ExpiresAt
		s.Candidate.VerifiedAt = renew.FetchedAt
	}
	if bad || renew != nil {
		if e = putSession(ctx, tx, s); e != nil {
			return s, e
		}
	}
	return s, tx.Commit()
}
