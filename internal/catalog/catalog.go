package catalog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Movie struct {
	ID     int      `json:"id"`
	Title  string   `json:"title"`
	Genres []string `json:"genres"`
	Mean   float64  `json:"mean"`
	Count  int      `json:"count"`
}
type Rating struct {
	User, Movie int
	Value       float64
	Time        int64
}
type Metadata struct {
	Source        string `json:"source"`
	Synthetic     bool   `json:"synthetic"`
	Movies        int    `json:"movies"`
	Ratings       int    `json:"ratings"`
	Users         int    `json:"users"`
	TrainRatings  int    `json:"train_ratings"`
	TestRatings   int    `json:"test_ratings"`
	Cutoff        int64  `json:"cutoff_unix"`
	RatingsSHA256 string `json:"ratings_sha256"`
	MoviesSHA256  string `json:"movies_sha256"`
}
type Catalog struct {
	Movies     map[int]Movie
	IDs        []int
	Train      map[int][]Rating
	Test       map[int][]Rating
	ByMovie    map[int][]Rating
	UserMean   map[int]float64
	GlobalMean float64
	Meta       Metadata
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func latin1(s string) string {
	var b strings.Builder
	for _, v := range []byte(s) {
		b.WriteRune(rune(v))
	}
	return b.String()
}

// Load uses a single global chronological split. No rating later than Cutoff
// contributes to popularity, user profiles, neighbor selection, or predictions.
func Load(dir string, trainFraction float64) (*Catalog, error) {
	if trainFraction <= 0 || trainFraction >= 1 {
		return nil, fmt.Errorf("train fraction must be between 0 and 1")
	}
	c := &Catalog{Movies: map[int]Movie{}}
	path := filepath.Join(dir, "movies.dat")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("MovieLens missing: run go run ./cmd/fetch-data: %w", err)
	}
	s := bufio.NewScanner(f)
	line := 0
	for s.Scan() {
		line++
		parts := strings.Split(latin1(s.Text()), "::")
		if len(parts) != 3 {
			f.Close()
			return nil, fmt.Errorf("movies line %d invalid", line)
		}
		id, e := strconv.Atoi(parts[0])
		if e != nil || id <= 0 {
			f.Close()
			return nil, fmt.Errorf("movies line %d invalid id", line)
		}
		if _, exists := c.Movies[id]; exists {
			f.Close()
			return nil, fmt.Errorf("duplicate movie %d", id)
		}
		c.Movies[id] = Movie{ID: id, Title: parts[1], Genres: strings.Split(parts[2], "|")}
	}
	err = s.Err()
	f.Close()
	if err != nil {
		return nil, err
	}
	f, err = os.Open(filepath.Join(dir, "ratings.dat"))
	if err != nil {
		return nil, err
	}
	s = bufio.NewScanner(f)
	ratings := make([]Rating, 0, 1000210)
	line = 0
	for s.Scan() {
		line++
		p := strings.Split(s.Text(), "::")
		if len(p) != 4 {
			f.Close()
			return nil, fmt.Errorf("ratings line %d invalid", line)
		}
		u, e1 := strconv.Atoi(p[0])
		m, e2 := strconv.Atoi(p[1])
		v, e3 := strconv.ParseFloat(p[2], 64)
		ts, e4 := strconv.ParseInt(p[3], 10, 64)
		_, exists := c.Movies[m]
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || u <= 0 || !exists || v < 1 || v > 5 || ts <= 0 {
			f.Close()
			return nil, fmt.Errorf("ratings line %d invalid values", line)
		}
		ratings = append(ratings, Rating{u, m, v, ts})
	}
	err = s.Err()
	f.Close()
	if err != nil {
		return nil, err
	}
	if len(ratings) < 100 {
		return nil, fmt.Errorf("expected full dataset, got %d ratings", len(ratings))
	}
	sort.Slice(ratings, func(i, j int) bool {
		if ratings[i].Time == ratings[j].Time {
			if ratings[i].User == ratings[j].User {
				return ratings[i].Movie < ratings[j].Movie
			}
			return ratings[i].User < ratings[j].User
		}
		return ratings[i].Time < ratings[j].Time
	})
	cutoff := ratings[int(float64(len(ratings))*trainFraction)-1].Time
	c.build(ratings, cutoff)
	c.Meta.Source = "MovieLens 1M / GroupLens (noncommercial research prototype)"
	c.Meta.RatingsSHA256, err = hashFile(filepath.Join(dir, "ratings.dat"))
	if err != nil {
		return nil, err
	}
	c.Meta.MoviesSHA256, err = hashFile(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalog) build(ratings []Rating, cutoff int64) {
	c.Train = map[int][]Rating{}
	c.Test = map[int][]Rating{}
	c.ByMovie = map[int][]Rating{}
	c.UserMean = map[int]float64{}
	c.Meta = Metadata{Movies: len(c.Movies), Ratings: len(ratings), Cutoff: cutoff}
	users := map[int]bool{}
	sum := 0.0
	for _, r := range ratings {
		users[r.User] = true
		if r.Time > cutoff {
			c.Test[r.User] = append(c.Test[r.User], r)
			c.Meta.TestRatings++
			continue
		}
		c.Train[r.User] = append(c.Train[r.User], r)
		c.ByMovie[r.Movie] = append(c.ByMovie[r.Movie], r)
		sum += r.Value
		c.Meta.TrainRatings++
	}
	c.Meta.Users = len(users)
	if c.Meta.TrainRatings > 0 {
		c.GlobalMean = sum / float64(c.Meta.TrainRatings)
	} else {
		c.GlobalMean = 3
	}
	for id, rs := range c.Train {
		sum := 0.0
		for _, r := range rs {
			sum += r.Value
		}
		c.UserMean[id] = sum / float64(len(rs))
	}
	for id, m := range c.Movies {
		rs := c.ByMovie[id]
		sum := 0.0
		for _, r := range rs {
			sum += r.Value
		}
		m.Count = len(rs)
		m.Mean = (sum + 25*c.GlobalMean) / (float64(len(rs)) + 25)
		c.Movies[id] = m
		c.IDs = append(c.IDs, id)
	}
	sort.Ints(c.IDs)
}

// Fixture is deliberately invented and only used by tests or explicit -demo.
func Fixture() *Catalog {
	c := &Catalog{Movies: map[int]Movie{}}
	genres := [][]string{{"Sci-Fi", "Adventure"}, {"Comedy", "Romance"}, {"Drama", "Comedy"}, {"Horror", "Thriller"}, {"Animation", "Comedy"}, {"Drama", "Sci-Fi"}, {"Action", "Adventure"}, {"Mystery", "Drama"}, {"Comedy"}, {"Sci-Fi", "Comedy"}, {"Documentary"}, {"Drama", "Romance"}}
	for i, g := range genres {
		id := i + 1
		c.Movies[id] = Movie{ID: id, Title: fmt.Sprintf("Invented Film %02d", id), Genres: g}
	}
	rs := []Rating{}
	for u := 1; u <= 12; u++ {
		for m := 1; m <= 12; m++ {
			rs = append(rs, Rating{u, m, float64(1 + (u+m)%5), int64(100 + m)})
		}
	}
	c.build(rs, 110)
	c.Meta.Synthetic = true
	c.Meta.Source = "invented test catalog (not MovieLens)"
	return c
}

type neighbor struct {
	user   int
	weight float64
}

// Predict returns a training-only user-user centered-cosine estimate. Similarity
// requires at least 4 common ratings and is shrunk by n/(n+20).
func (c *Catalog) Predict(user int) map[int]float64 {
	base := c.UserMean[user]
	if base == 0 {
		base = c.GlobalMean
	}
	type accum struct {
		dot, a, b float64
		n         int
	}
	stats := map[int]*accum{}
	for _, r := range c.Train[user] {
		x := r.Value - base
		for _, other := range c.ByMovie[r.Movie] {
			if other.User == user {
				continue
			}
			a := stats[other.User]
			if a == nil {
				a = &accum{}
				stats[other.User] = a
			}
			y := other.Value - c.UserMean[other.User]
			a.dot += x * y
			a.a += x * x
			a.b += y * y
			a.n++
		}
	}
	neighbors := []neighbor{}
	for u, a := range stats {
		if a.n < 4 || a.a == 0 || a.b == 0 {
			continue
		}
		w := a.dot / math.Sqrt(a.a*a.b) * float64(a.n) / float64(a.n+20)
		if w > 0 {
			neighbors = append(neighbors, neighbor{u, w})
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		if neighbors[i].weight == neighbors[j].weight {
			return neighbors[i].user < neighbors[j].user
		}
		return neighbors[i].weight > neighbors[j].weight
	})
	if len(neighbors) > 40 {
		neighbors = neighbors[:40]
	}
	num, den := map[int]float64{}, map[int]float64{}
	for _, n := range neighbors {
		for _, r := range c.Train[n.user] {
			num[r.Movie] += n.weight * (r.Value - c.UserMean[n.user])
			den[r.Movie] += n.weight
		}
	}
	pred := make(map[int]float64, len(c.Movies))
	for id, m := range c.Movies {
		v := m.Mean + 0.35*(base-c.GlobalMean)
		if den[id] > 0 {
			confidence := den[id] / (den[id] + 2)
			v = confidence*(base+num[id]/den[id]) + (1-confidence)*v
		}
		pred[id] = math.Max(1, math.Min(5, v))
	}
	return pred
}

func (c *Catalog) Profiles() []map[string]any {
	ids := []int{}
	for id, rs := range c.Train {
		if len(rs) >= 20 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	// Fixed sampling avoids selecting only unusually generous first-ID raters.
	// Demo profiles are distinct from the evaluation sample.
	rng := rand.New(rand.NewSource(20261013))
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	if len(ids) > 30 {
		ids = ids[:30]
	}
	out := []map[string]any{}
	for _, id := range ids {
		rs := c.Train[id]
		out = append(out, map[string]any{"id": id, "ratings": len(rs), "mean": c.UserMean[id]})
	}
	return out
}
