package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"commonreel/internal/catalog"
	"commonreel/internal/recommend"
	"commonreel/internal/room"
)

type metrics struct {
	Recall float64 `json:"recall_at_10"`
	NDCG   float64 `json:"ndcg_at_10"`
	Hit    float64 `json:"hit_rate_at_10"`
}

func measure(ranked []int, relevant map[int]bool) metrics {
	hits, dcg := 0, 0.0
	for i, id := range ranked {
		if i >= 10 {
			break
		}
		if relevant[id] {
			hits++
			dcg += 1 / math.Log2(float64(i+2))
		}
	}
	ideal := 0.0
	for i := 0; i < len(relevant) && i < 10; i++ {
		ideal += 1 / math.Log2(float64(i+2))
	}
	hit := 0.0
	if hits > 0 {
		hit = 1
	}
	return metrics{float64(hits) / float64(len(relevant)), dcg / ideal, hit}
}
func rank(c *catalog.Catalog, ids []int, pred map[int]float64) []int {
	out := append([]int{}, ids...)
	sort.Slice(out, func(i, j int) bool {
		a, b := pred[out[i]], pred[out[j]]
		if a == b {
			return out[i] < out[j]
		}
		return a > b
	})
	return out
}
func main() {
	dir := flag.String("data", "data/ml-1m", "MovieLens data directory")
	count := flag.Int("users", 200, "sample size")
	output := flag.String("out", "docs/evidence/evaluation.json", "report file")
	flag.Parse()
	if *count < 1 {
		log.Fatal("users must be positive")
	}
	started := time.Now()
	c, err := catalog.Load(*dir, 0.8)
	if err != nil {
		log.Fatal(err)
	}
	eligible := []int{}
	positives := map[int]map[int]bool{}
	candidates := map[int][]int{}
	for u, test := range c.Test {
		if len(c.Train[u]) < 20 {
			continue
		}
		seen := map[int]bool{}
		for _, r := range c.Train[u] {
			seen[r.Movie] = true
		}
		positive := map[int]bool{}
		for _, r := range test {
			if r.Value >= 4 && !seen[r.Movie] && c.Movies[r.Movie].Count >= 10 {
				positive[r.Movie] = true
			}
		}
		if len(positive) < 5 {
			continue
		}
		ids := []int{}
		for _, id := range c.IDs {
			if !seen[id] && c.Movies[id].Count >= 10 {
				ids = append(ids, id)
			}
		}
		eligible = append(eligible, u)
		positives[u] = positive
		candidates[u] = ids
	}
	sort.Ints(eligible)
	rng := rand.New(rand.NewSource(20261013))
	rng.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
	totalEligible := len(eligible)
	if len(eligible) > *count {
		eligible = eligible[:*count]
	}
	if len(eligible) == 0 {
		log.Fatal("no eligible holdout users")
	}
	pop := map[int]float64{}
	for id, m := range c.Movies {
		pop[id] = m.Mean
	}
	popular, personal := metrics{}, metrics{}
	minCandidates, maxCandidates := math.MaxInt, 0
	totalPositives := 0
	for _, u := range eligible {
		p := measure(rank(c, candidates[u], pop), positives[u])
		q := measure(rank(c, candidates[u], c.Predict(u)), positives[u])
		popular.Recall += p.Recall
		popular.NDCG += p.NDCG
		popular.Hit += p.Hit
		personal.Recall += q.Recall
		personal.NDCG += q.NDCG
		personal.Hit += q.Hit
		n := len(candidates[u])
		if n < minCandidates {
			minCandidates = n
		}
		if n > maxCandidates {
			maxCandidates = n
		}
		totalPositives += len(positives[u])
	}
	n := float64(len(eligible))
	popular.Recall /= n
	popular.NDCG /= n
	popular.Hit /= n
	personal.Recall /= n
	personal.NDCG /= n
	personal.Hit /= n
	e := recommend.New(c)
	profiles := c.Profiles()
	if len(profiles) < 3 {
		log.Fatal("need three training profiles")
	}
	profileIDs := []int{profiles[0]["id"].(int), profiles[1]["id"].(int), profiles[2]["id"].(int)}
	demo := room.Room{Revision: 1, ExcludeSeen: true, Members: []room.Member{{ID: "a", Name: "Alex", Profile: profileIDs[0], Likes: []string{"Sci-Fi", "Adventure"}}, {ID: "b", Name: "Jamie", Profile: profileIDs[1], Likes: []string{"Comedy", "Romance"}, Vetoes: []string{"Horror"}}, {ID: "c", Name: "Sam", Profile: profileIDs[2], Likes: []string{"Drama"}}}}
	coldStart := time.Now()
	first := e.Shortlist(demo, 6, "fair")
	coldMS := float64(time.Since(coldStart).Microseconds()) / 1000
	durations := []float64{}
	vetoViolations := 0
	for i := 0; i < 100; i++ {
		start := time.Now()
		result := e.Shortlist(demo, 6, "fair")
		durations = append(durations, float64(time.Since(start).Microseconds())/1000)
		for _, v := range result.Candidates {
			for _, g := range v.Movie.Genres {
				if g == "Horror" {
					vetoViolations++
				}
			}
		}
	}
	sort.Float64s(durations)
	out := map[string]any{
		"run_utc": time.Now().UTC().Format(time.RFC3339), "dataset": c.Meta, "runtime": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "logical_cpus": runtime.NumCPU(), "elapsed_seconds": time.Since(started).Seconds(),
		"protocol":       "single global chronological 80/20 timestamp split; ties stay in training; relevance=heldout rating>=4; full unseen training-vocabulary candidate set; minimum 20 training ratings and 5 eligible heldout positives; seed=20261013; deterministic top-10 ties by movie ID",
		"eligible_users": totalEligible, "evaluated_users": len(eligible), "user_ids": eligible, "heldout_positive_count": totalPositives, "candidate_count_min": minCandidates, "candidate_count_max": maxCandidates,
		"popularity_baseline": popular, "centered_user_knn": personal,
		"performance":       map[string]any{"measurement": "in-process shortlist, warmed cache, three fixed replay profiles, sequential 100 calls, excludes HTTP/MCP overhead", "cold_ms": coldMS, "warm_p50_ms": durations[49], "warm_p95_ms": durations[94], "warm_max_ms": durations[99], "iterations": 100, "shortlist_size": len(first.Candidates)},
		"constraint_replay": map[string]any{"nature": "constructed 3-person group using research histories; not real group feedback", "profile_ids": profileIDs, "returned_candidates_total": 100 * len(first.Candidates), "horror_veto_violations": vetoViolations},
		"limits":            []string{"2000-era MovieLens ratings; no modern streaming catalog", "offline individual ranking does not measure group satisfaction", "genre preference overrides and synthetic grouping are not heldout labels", "latency is local in-process and is not concurrent production throughput", "no human study or live Alexa+ device validation"},
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	if err = os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(*output, b, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
