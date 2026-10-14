package recommend

import (
	"commonreel/internal/catalog"
	"commonreel/internal/room"
	"testing"
)

func TestVetoesAndEmptyResult(t *testing.T) {
	c := catalog.Fixture()
	e := New(c)
	r := room.Room{Revision: 3, Members: []room.Member{{ID: "a", Name: "A", Profile: 1, Vetoes: []string{"Horror"}}, {ID: "b", Name: "B", Profile: 2, Likes: []string{"Drama"}}}}
	result := e.Shortlist(r, 30, "fair")
	if len(result.Candidates) == 0 {
		t.Fatal("expected candidates")
	}
	for _, v := range result.Candidates {
		for _, g := range v.Movie.Genres {
			if g == "Horror" {
				t.Fatal("veto violated")
			}
		}
		if v.Minimum > v.Average {
			t.Fatal("invalid group floor")
		}
		if len(v.Matches) != 2 {
			t.Fatal("missing member explanation")
		}
	}
	all := map[string]bool{}
	for _, m := range c.Movies {
		for _, g := range m.Genres {
			all[g] = true
		}
	}
	r.Members[0].Vetoes = nil
	for g := range all {
		r.Members[0].Vetoes = append(r.Members[0].Vetoes, g)
	}
	result = e.Shortlist(r, 30, "fair")
	if len(result.Candidates) != 0 || result.Message == "" {
		t.Fatal("impossible constraints were relaxed")
	}
}
func TestRejectAmbiguousPreferences(t *testing.T) {
	e := New(catalog.Fixture())
	if err := e.Validate(room.Member{Name: "A", Likes: []string{"Horror"}, Vetoes: []string{"Horror"}}); err == nil {
		t.Fatal("contradiction accepted")
	}
	if err := e.Validate(room.Member{Name: "A", Profile: 999}); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
