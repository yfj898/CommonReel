package catalog

import (
	"reflect"
	"testing"
)

func TestFutureRatingsCannotAffectTrainingPredictions(t *testing.T) {
	a := Fixture()
	b := &Catalog{Movies: map[int]Movie{}}
	for id, m := range a.Movies {
		m.Count = 0
		m.Mean = 0
		b.Movies[id] = m
	}
	rs := []Rating{}
	for _, user := range a.Train {
		rs = append(rs, user...)
	}
	for _, user := range a.Test {
		for _, r := range user {
			r.Value = 6 - r.Value
			rs = append(rs, r)
		}
	}
	b.build(rs, a.Meta.Cutoff)
	if a.GlobalMean != b.GlobalMean {
		t.Fatal("future ratings affected prior")
	}
	for id, m := range a.Movies {
		if m.Mean != b.Movies[id].Mean {
			t.Fatal("future ratings affected movie prior")
		}
	}
	if !reflect.DeepEqual(a.Predict(1), b.Predict(1)) {
		t.Fatal("future ratings leaked into collaborative predictions")
	}
}
