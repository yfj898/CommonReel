package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Film struct {
	ID              string   `json:"film_id"`
	Title           string   `json:"title"`
	Year            int      `json:"year"`
	Aliases         []string `json:"aliases,omitempty"`
	IMDb            string   `json:"imdb_id,omitempty"`
	Watchmode       int      `json:"watchmode_id,omitempty"`
	Runtime         int      `json:"runtime_minutes"`
	RuntimeConflict bool     `json:"runtime_conflict,omitempty"`
	Genres          []string `json:"genres"`
	Prior           float64  `json:"catalog_prior"`
	Version         string   `json:"catalog_version"`
	Source          string   `json:"source"`
}
type Offer struct {
	Name         string `json:"name,omitempty"`
	Region       string `json:"region"`
	ProviderID   int    `json:"provider_id"`
	Monetization string `json:"monetization"`
	URL          string `json:"url,omitempty"`
}
type Availability struct {
	FilmID    string    `json:"film_id"`
	Known     bool      `json:"known"`
	Offers    []Offer   `json:"offers"`
	FetchedAt time.Time `json:"fetched_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Version   string    `json:"facts_version"`
	Reason    string    `json:"reason,omitempty"`
}
type Supplier interface {
	Fetch(context.Context, Film) (Availability, error)
}
type Policy struct {
	MaxRuntime  int   `json:"max_runtime_minutes"`
	ProviderIDs []int `json:"provider_ids"`
	Rewatch     bool  `json:"allow_rewatch"`
}
type Preferences struct {
	LikedGenres []string           `json:"liked_genres"`
	Vetoes      []string           `json:"vetoes"`
	Seen        []string           `json:"seen"`
	Ratings     map[string]float64 `json:"ratings"`
	Likes       map[string]bool    `json:"likes"`
}
type PreferencePatch struct {
	SeenAdd     []string            `json:"seen_add,omitempty"`
	LikedGenres *[]string           `json:"liked_genres,omitempty"`
	Vetoes      *[]string           `json:"vetoes,omitempty"`
	Seen        *[]string           `json:"seen,omitempty"`
	Ratings     *map[string]float64 `json:"ratings,omitempty"`
	Likes       *map[string]bool    `json:"likes,omitempty"`
}
type ActiveCandidate struct {
	Title          string    `json:"title"`
	FilmID         string    `json:"film_id"`
	Decision       int       `json:"decision_version"`
	CatalogVersion string    `json:"catalog_version"`
	FactsVersion   string    `json:"facts_version"`
	VerifiedAt     time.Time `json:"verified_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}
type Acceptance struct {
	MemberID string    `json:"target_member_id"`
	FilmID   string    `json:"film_id"`
	Decision int       `json:"decision_version"`
	Response string    `json:"response"`
	Actor    string    `json:"actor_member_id"`
	At       time.Time `json:"at"`
}
type ChoiceSummary struct {
	ID           string    `json:"id"`
	FilmID       string    `json:"film_id"`
	At           time.Time `json:"saved_at"`
	Policy       Policy    `json:"policy"`
	FactsVersion string    `json:"facts_version"`
	Decision     int       `json:"decision_version"`
	Label        string    `json:"label"`
}
type Candidate struct {
	PreferenceBasis string             `json:"preference_basis"`
	EstimateBasis   map[string]string  `json:"estimate_basis"`
	Film            Film               `json:"film"`
	Availability    Availability       `json:"availability"`
	Score           float64            `json:"estimated_group_score"`
	Minimum         float64            `json:"estimated_minimum"`
	Mean            float64            `json:"estimated_mean"`
	Estimates       map[string]float64 `json:"estimates"`
}
type Shortlist struct {
	CatalogTotal   int            `json:"catalog_total"`
	SupplyVerified int            `json:"supply_verified"`
	Unknown        int            `json:"unknown_excluded"`
	ProbeLimit     int            `json:"probe_limit"`
	Candidates     []Candidate    `json:"candidates"`
	Filtered       map[string]int `json:"filtered"`
	Mode           string         `json:"mode"`
	Decision       int            `json:"decision_version"`
}

func memberIndex(s Session, id string) int {
	for i, m := range s.Members {
		if m.ID == id {
			return i
		}
	}
	return -1
}
func contains[T comparable](xs []T, x T) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func (a *App) film(id string) (Film, bool) {
	for _, f := range a.cfg.Films {
		if f.ID == id {
			return f, true
		}
	}
	return Film{}, false
}
func patchPreferences(p *Preferences, x PreferencePatch, a *App) error {
	if x.Seen != nil {
		ids := []string{}
		for _, id := range *x.Seen {
			canonical, e := a.canonicalFilm(id)
			if e != nil {
				return e
			}
			ids = append(ids, canonical)
		}
		x.Seen = &ids
	}
	for i, id := range x.SeenAdd {
		canonical, e := a.canonicalFilm(id)
		if e != nil {
			return e
		}
		x.SeenAdd[i] = canonical
	}
	if x.LikedGenres != nil {
		p.LikedGenres = *x.LikedGenres
	}
	if x.Vetoes != nil {
		p.Vetoes = *x.Vetoes
	}
	if x.Seen != nil {
		if len(*x.Seen) > 50000 {
			return fail("INVALID_ARGUMENT", 400, "too many seen films")
		}
		p.Seen = *x.Seen
	}
	if x.Ratings != nil {
		for _, v := range *x.Ratings {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 1 || v > 5 {
				return fail("INVALID_ARGUMENT", 400, "ratings must be 1-5")
			}
		}
		if p.Ratings == nil {
			p.Ratings = map[string]float64{}
		}
		for k, v := range *x.Ratings {
			canonical, e := a.canonicalFilm(k)
			if e != nil {
				return e
			}
			p.Ratings[canonical] = v
		}
	}
	if x.Likes != nil {
		if p.Likes == nil {
			p.Likes = map[string]bool{}
		}
		for k, v := range *x.Likes {
			canonical, e := a.canonicalFilm(k)
			if e != nil {
				return e
			}
			p.Likes[canonical] = v
		}
	}
	if len(x.SeenAdd) > 0 {
		p.Seen = unique(append(p.Seen, x.SeenAdd...))
	}
	canonical := map[string]string{}
	for _, g := range []string{"Comedy", "Drama", "Horror", "Thriller", "Sci-Fi", "Fantasy", "Romance", "Action", "Adventure", "Animation", "Documentary", "Crime", "Mystery", "War", "Western", "Musical", "Family"} {
		canonical[strings.ToLower(g)] = g
	}
	for _, f := range a.cfg.Films {
		for _, g := range f.Genres {
			canonical[strings.ToLower(g)] = g
		}
	}
	for _, xs := range []*[]string{&p.LikedGenres, &p.Vetoes} {
		normalized := []string{}
		for _, g := range *xs {
			v, ok := canonical[strings.ToLower(strings.TrimSpace(g))]
			if !ok {
				return fail("INVALID_ARGUMENT", 400, "unknown genre; use labels shown in the catalog")
			}
			normalized = append(normalized, v)
		}
		*xs = normalized
	}
	for _, xs := range [][]string{p.LikedGenres, p.Vetoes, p.Seen} {
		sort.Strings(xs)
	}
	p.LikedGenres = unique(p.LikedGenres)
	p.Vetoes = unique(p.Vetoes)
	p.Seen = unique(p.Seen)
	return nil
}

var externalMovieID = regexp.MustCompile(`^imdb:tt[0-9]{7,10}$`)

func (a *App) canonicalFilm(id string) (string, error) {
	if _, ok := a.film(id); ok {
		return id, nil
	}
	if externalMovieID.MatchString(id) {
		for _, f := range a.cfg.Films {
			if f.IMDb == strings.TrimPrefix(id, "imdb:") {
				return f.ID, nil
			}
		}
		return id, nil
	}
	return "", fail("INVALID_ARGUMENT", 400, "choose a supported film ID or an explicitly confirmed imdb:tt identifier")
}

func (a *App) availability(ctx context.Context, f Film, refresh bool) Availability {
	a.cacheMu.Lock()
	old, ok := a.cache[f.ID]
	a.cacheMu.Unlock()
	if !refresh && ok && a.cfg.Now().Before(old.ExpiresAt) {
		return old
	}
	av := Availability{FilmID: f.ID, Reason: "not_queried"}
	if a.cfg.Supplier != nil {
		select {
		case a.supplySlots <- struct{}{}:
			defer func() { <-a.supplySlots }()
		case <-ctx.Done():
			return av
		}
		v, e := a.cfg.Supplier.Fetch(ctx, f)
		av = v
		av.FilmID = f.ID
		if e != nil {
			av.Known = false
			av.Reason = "supplier_unavailable"
		}
	}
	if !av.Known {
		av.FetchedAt = a.cfg.Now().UTC()
		av.ExpiresAt = av.FetchedAt.Add(time.Minute)
	}
	sort.Slice(av.Offers, func(i, j int) bool { return hashRequest(av.Offers[i]) < hashRequest(av.Offers[j]) })
	av.Version = availabilityVersion(av)
	a.cacheMu.Lock()
	a.cache[f.ID] = av
	a.cacheMu.Unlock()
	a.persistSupply(ctx, av)
	return av
}
func availabilityVersion(av Availability) string {
	facts := []string{}
	for _, o := range av.Offers {
		facts = append(facts, fmt.Sprintf("%s:%d:%s", o.Region, o.ProviderID, o.Monetization))
	}
	facts = unique(facts)
	sort.Strings(facts)
	return hashRequest(struct {
		Known bool
		Facts []string
	}{av.Known, facts})
}
func (a *App) eligibility(s Session, f Film, av Availability) string {
	if len(s.Members) < 2 || len(s.Members) > 4 || s.Policy.MaxRuntime < 1 || len(s.Policy.ProviderIDs) == 0 {
		return "incomplete_group_or_policy"
	}
	if f.Runtime <= 0 || f.RuntimeConflict || len(f.Genres) == 0 {
		return "metadata_unknown"
	}
	if f.Runtime > s.Policy.MaxRuntime {
		return "runtime"
	}
	for _, m := range s.Members {
		for _, g := range f.Genres {
			if contains(m.Preferences.Vetoes, g) {
				return "genre_veto"
			}
		}
		if !s.Policy.Rewatch && (contains(effectiveSeen(m), f.ID) || f.IMDb != "" && contains(effectiveSeen(m), "imdb:"+f.IMDb)) {
			return "already_seen"
		}
	}
	if !av.Known || !a.cfg.Now().Before(av.ExpiresAt) {
		return "availability_unknown"
	}
	for _, o := range av.Offers {
		if o.Region == "US" && o.Monetization == "sub" && contains(s.Policy.ProviderIDs, o.ProviderID) {
			return ""
		}
	}
	return "no_selected_subscription"
}
func explicitRating(m Member, f Film) (float64, bool) {
	if v, ok := m.Preferences.Ratings[f.ID]; ok {
		return v, true
	}
	if f.IMDb != "" {
		v, ok := m.Preferences.Ratings["imdb:"+f.IMDb]
		return v, ok
	}
	return 0, false
}
func explicitlyLiked(m Member, f Film) bool {
	if liked, ok := m.Preferences.Likes[f.ID]; ok {
		return liked
	}
	return f.IMDb != "" && m.Preferences.Likes["imdb:"+f.IMDb]
}
func estimate(m Member, f Film, films []Film) (float64, string) {
	if v, ok := explicitRating(m, f); ok {
		return v, "explicit_rating"
	}
	prior := f.Prior
	if prior < 1 || prior > 5 {
		prior = 3
	}
	for _, g := range f.Genres {
		if contains(m.Preferences.LikedGenres, g) {
			return math.Min(5, prior+0.3), "liked_genre_match"
		}
	}
	for _, liked := range films {
		rating, _ := explicitRating(m, liked)
		if explicitlyLiked(m, liked) || rating >= 4 {
			for _, g := range f.Genres {
				if contains(liked.Genres, g) {
					return math.Min(5, prior+0.3), "liked_film_genre_match"
				}
			}
		}
	}
	return prior, "catalog_baseline"
}

func (a *App) shortlist(ctx context.Context, id, secret, mode string) (Shortlist, error) {
	s, e := a.current(ctx, id, secret)
	if e != nil {
		return Shortlist{}, e
	}
	if mode == "" {
		mode = "fair"
	}
	if mode != "fair" && mode != "average" && mode != "catalog_prior" {
		return Shortlist{}, fail("INVALID_ARGUMENT", 400, "unknown ranking mode")
	}
	result := Shortlist{CatalogTotal: len(a.cfg.Films), ProbeLimit: 30, Candidates: []Candidate{}, Filtered: map[string]int{}, Mode: mode, Decision: s.Decision}
	// One bounded round; multiple rooms still share the application's four slots.
	probeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var workers sync.WaitGroup
	scheduled := 0
	for _, f := range a.cfg.Films {
		dummy := Availability{Known: true, ExpiresAt: a.cfg.Now().Add(time.Hour), Offers: []Offer{{Region: "US", Monetization: "sub", ProviderID: firstProvider(s.Policy)}}}
		if a.eligibility(s, f, dummy) != "" {
			continue
		}
		a.cacheMu.Lock()
		old, ok := a.cache[f.ID]
		a.cacheMu.Unlock()
		if ok && a.cfg.Now().Before(old.ExpiresAt) {
			continue
		}
		if scheduled >= 30 {
			break
		}
		scheduled++
		workers.Add(1)
		go func(f Film) { defer workers.Done(); a.availability(probeCtx, f, false) }(f)
	}
	workers.Wait()
	for _, f := range a.cfg.Films {
		// Hard metadata/member failures are determined before spending supplier credits.
		dummy := Availability{Known: true, ExpiresAt: a.cfg.Now().Add(time.Hour), Offers: []Offer{{Region: "US", Monetization: "sub", ProviderID: firstProvider(s.Policy)}}}
		if reason := a.eligibility(s, f, dummy); reason != "" {
			result.Filtered[reason]++
			continue
		}
		a.cacheMu.Lock()
		av, ok := a.cache[f.ID]
		a.cacheMu.Unlock()
		if !ok || !a.cfg.Now().Before(av.ExpiresAt) {
			av = Availability{}
		}
		if av.Known && a.cfg.Now().Before(av.ExpiresAt) {
			result.SupplyVerified++
		}
		reason := a.eligibility(s, f, av)
		if reason != "" {
			result.Filtered[reason]++
			if reason == "availability_unknown" {
				result.Unknown++
			}
			continue
		}
		c := a.scoreCandidate(s, f, av, mode)
		result.Candidates = append(result.Candidates, c)
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Score == result.Candidates[j].Score {
			return result.Candidates[i].Film.ID < result.Candidates[j].Film.ID
		}
		return result.Candidates[i].Score > result.Candidates[j].Score
	})
	if len(result.Candidates) > 6 {
		result.Candidates = result.Candidates[:6]
	}
	return result, nil
}
func firstProvider(p Policy) int {
	if len(p.ProviderIDs) > 0 {
		return p.ProviderIDs[0]
	}
	return 0
}
func readyToSave(s Session, c Command) error {
	if len(s.Members) < 2 || s.Candidate == nil || s.Candidate.FilmID != c.FilmID || s.Decision != c.Decision {
		return fail("SELECTION_STALE", 409, "candidate or version changed")
	}
	for _, m := range s.Members {
		v, ok := s.Acceptances[m.ID]
		if !ok || v.Response != "accept" || v.FilmID != c.FilmID || v.Decision != s.Decision {
			return fail("ACCEPTANCE_INCOMPLETE", 409, "all member entries must record acceptance")
		}
	}
	return nil
}
func (a *App) invalidateSelection(ctx context.Context, id, secret string, rev int) (Receipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return Receipt{}, e
	}
	defer tx.Rollback()
	s, e := a.authorized(ctx, tx, id, secret)
	if e != nil {
		return Receipt{}, e
	}
	if s.Revision != rev {
		return Receipt{}, fail("REVISION_CONFLICT", 409, "conditions changed during refresh")
	}
	s.advanceDecision(nil)
	if e = putSession(ctx, tx, s); e != nil {
		return Receipt{}, e
	}
	if e = tx.Commit(); e != nil {
		return Receipt{}, e
	}
	return Receipt{}, fail("SELECTION_STALE", 409, "eligibility facts changed; propose again and record responses")
}
func (a *App) catalogRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sessions/{id}/shortlist", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.shortlist(r.Context(), r.PathValue("id"), token(r), r.URL.Query().Get("mode"))
		if e != nil {
			writeError(w, e)
			return
		}
		write(w, 200, v)
	})
	mux.HandleFunc("GET /api/v1/choice-summaries/{id}", func(w http.ResponseWriter, r *http.Request) {
		var hash, raw string
		e := a.db.QueryRowContext(r.Context(), "SELECT capability_hash,payload FROM v2_summaries WHERE id=?", r.PathValue("id")).Scan(&hash, &raw)
		if e != nil {
			writeError(w, fail("NOT_FOUND", 404, "summary not found"))
			return
		}
		if digest(token(r)) != hash {
			writeError(w, fail("UNAUTHORIZED", 401, "invalid room capability"))
			return
		}
		var v any
		json.Unmarshal([]byte(raw), &v)
		write(w, 200, v)
	})
	mux.HandleFunc("GET /api/v1/catalog", func(w http.ResponseWriter, r *http.Request) {
		films := []Film{}
		q := strings.ToLower(r.URL.Query().Get("q"))
		ids := map[string]bool{}
		if raw := r.URL.Query().Get("ids"); raw != "" {
			parts := strings.Split(raw, ",")
			if len(parts) > 100 {
				writeError(w, fail("INVALID_ARGUMENT", 400, "request at most 100 film identifiers"))
				return
			}
			for _, id := range parts {
				ids[strings.TrimSpace(id)] = true
			}
		}
		for _, f := range a.cfg.Films {
			if (len(ids) == 0 || ids[f.ID] || (f.IMDb != "" && ids["imdb:"+f.IMDb])) && (q == "" || strings.Contains(strings.ToLower(f.Title), q)) {
				films = append(films, f)
			}
			if len(films) >= 100 {
				break
			}
		}
		write(w, 200, map[string]any{"catalog_total": len(a.cfg.Films), "films": films, "coverage": "bounded movie catalog; US subscription supply is checked separately", "attribution": "Streaming availability powered by Watchmode (https://www.watchmode.com/); metadata: Wikidata CC0"})
	})
}

func (a *App) scoreCandidate(s Session, f Film, av Availability, mode string) Candidate {
	c := Candidate{Film: f, Availability: av, Estimates: map[string]float64{}, EstimateBasis: map[string]string{}, PreferenceBasis: "catalog_baseline", Minimum: 5}
	explicit := 0
	for _, m := range s.Members {
		v, basis := estimate(m, f, a.cfg.Films)
		c.EstimateBasis[m.ID] = basis
		if basis != "catalog_baseline" {
			explicit++
		}
		c.Estimates[m.ID] = v
		c.Mean += v
		if v < c.Minimum {
			c.Minimum = v
		}
	}
	if explicit > 0 {
		c.PreferenceBasis = "partial_explicit"
		if explicit == len(s.Members) {
			c.PreferenceBasis = "explicit_preferences"
		}
	}
	c.Mean /= float64(len(s.Members))
	c.Score = .65*c.Minimum + .35*c.Mean
	if mode == "average" {
		c.Score = c.Mean
	}
	if mode == "catalog_prior" {
		c.Score = f.Prior
		if c.Score < 1 || c.Score > 5 {
			c.Score = 3
		}
	}
	return c
}
