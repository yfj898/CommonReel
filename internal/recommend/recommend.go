package recommend

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"commonreel/internal/catalog"
	"commonreel/internal/room"
)

type Match struct {
	MemberID string  `json:"member_id"`
	Name     string  `json:"name"`
	Estimate float64 `json:"estimate"`
	Reason   string  `json:"reason"`
}
type Candidate struct {
	Movie       catalog.Movie `json:"movie"`
	Score       float64       `json:"score"`
	Minimum     float64       `json:"minimum"`
	Average     float64       `json:"average"`
	Matches     []Match       `json:"matches"`
	Explanation string        `json:"explanation"`
}
type Result struct {
	RoomRevision int              `json:"room_revision"`
	Candidates   []Candidate      `json:"candidates"`
	Considered   int              `json:"considered"`
	Vetoed       int              `json:"vetoed"`
	Seen         int              `json:"seen"`
	Message      string           `json:"message"`
	Data         catalog.Metadata `json:"data"`
}
type Engine struct {
	Catalog *catalog.Catalog
	mu      sync.Mutex
	cache   map[int]map[int]float64
	order   []int
}

func New(c *catalog.Catalog) *Engine { return &Engine{Catalog: c, cache: map[int]map[int]float64{}} }
func (e *Engine) pred(u int) map[int]float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p := e.cache[u]; p != nil {
		return p
	}
	p := e.Catalog.Predict(u)
	if len(e.order) >= 64 {
		delete(e.cache, e.order[0])
		e.order = e.order[1:]
	}
	e.cache[u] = p
	e.order = append(e.order, u)
	return p
}
func (e *Engine) Validate(m room.Member) error {
	if strings.TrimSpace(m.Name) == "" || len(m.Name) > 48 {
		return fmt.Errorf("member name must contain 1–48 bytes")
	}
	if m.Profile != 0 && len(e.Catalog.Train[m.Profile]) == 0 {
		return fmt.Errorf("profile has no training ratings")
	}
	allowed := map[string]bool{}
	for _, movie := range e.Catalog.Movies {
		for _, g := range movie.Genres {
			allowed[g] = true
		}
	}
	if len(m.Likes) > 8 || len(m.Vetoes) > 8 {
		return fmt.Errorf("at most 8 genre preferences")
	}
	seen := map[string]bool{}
	for _, g := range append(append([]string{}, m.Likes...), m.Vetoes...) {
		if !allowed[g] {
			return fmt.Errorf("unknown genre %q", g)
		}
		if seen[g] {
			return fmt.Errorf("duplicate or contradictory genre %q", g)
		}
		seen[g] = true
	}
	return nil
}
func (e *Engine) Shortlist(r room.Room, limit int, mode string) Result {
	if limit < 1 {
		limit = 6
	}
	out := Result{RoomRevision: r.Revision, Candidates: []Candidate{}, Data: e.Catalog.Meta}
	if len(r.Members) == 0 {
		out.Message = "Add at least one person to build a shared shortlist."
		return out
	}
	predictions := map[string]map[int]float64{}
	seen := map[int]bool{}
	vetoes := map[string]bool{}
	for _, m := range r.Members {
		predictions[m.ID] = e.pred(m.Profile)
		for _, v := range m.Vetoes {
			vetoes[v] = true
		}
		if r.ExcludeSeen {
			for _, v := range e.Catalog.Train[m.Profile] {
				seen[v.Movie] = true
			}
		}
	}
	for _, id := range e.Catalog.IDs {
		movie := e.Catalog.Movies[id]
		if movie.Count < 10 {
			continue
		}
		out.Considered++
		blocked := false
		for _, g := range movie.Genres {
			if vetoes[g] {
				blocked = true
			}
		}
		if blocked {
			out.Vetoed++
			continue
		}
		if seen[id] {
			out.Seen++
			continue
		}
		candidate := Candidate{Movie: movie, Minimum: 5, Matches: []Match{}}
		for _, member := range r.Members {
			value := predictions[member.ID][id]
			reason := "training-only collaborative estimate"
			if member.Profile == 0 {
				reason = "catalog prior; no rating history"
			}
			hits := []string{}
			for _, g := range member.Likes {
				for _, mg := range movie.Genres {
					if g == mg {
						hits = append(hits, g)
					}
				}
			}
			if len(hits) > 0 {
				value = math.Min(5, value+0.3)
				reason += "; likes " + strings.Join(hits, ", ")
			}
			candidate.Matches = append(candidate.Matches, Match{member.ID, member.Name, value, reason})
			candidate.Average += value
			if value < candidate.Minimum {
				candidate.Minimum = value
			}
		}
		candidate.Average /= float64(len(r.Members))
		candidate.Score = 0.65*candidate.Minimum + 0.35*candidate.Average
		if mode == "average" {
			candidate.Score = candidate.Average
		}
		if mode == "popularity" {
			candidate.Score = movie.Mean
		}
		candidate.Explanation = fmt.Sprintf("Estimated floor %.2f/5, group average %.2f/5. All %d members' genre vetoes respected. Estimates are not guarantees of enjoyment.", candidate.Minimum, candidate.Average, len(r.Members))
		out.Candidates = append(out.Candidates, candidate)
	}
	sort.Slice(out.Candidates, func(i, j int) bool {
		if out.Candidates[i].Score == out.Candidates[j].Score {
			return out.Candidates[i].Movie.ID < out.Candidates[j].Movie.ID
		}
		return out.Candidates[i].Score > out.Candidates[j].Score
	})
	if len(out.Candidates) > limit {
		out.Candidates = out.Candidates[:limit]
	}
	if len(out.Candidates) == 0 {
		out.Message = "No films satisfy the current vetoes and unseen-only setting. Ask the group to change a constraint; no veto was silently relaxed."
	} else {
		out.Message = "A shortlist that keeps the least enthusiastic person in the conversation."
	}
	return out
}
