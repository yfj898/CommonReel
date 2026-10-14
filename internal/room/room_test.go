package room

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentRetryAndDurability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rooms.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	r, err := s.Create(ctx, "shared night")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			next, e := s.Mutate(ctx, r.ID, 1, "retry-key-0001", map[string]any{"action": "join"}, func(r *Room) error { r.Members = append(r.Members, Member{ID: ID(), Name: "Alex"}); return nil })
			if e != nil {
				failures <- e
			} else if next.Revision != 2 || len(next.Members) != 1 {
				failures <- errors.New("retry was not exactly once")
			}
		}()
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		t.Fatal(e)
	}
	if _, err = s.Mutate(ctx, r.ID, 1, "retry-key-0001", map[string]any{"action": "different"}, func(r *Room) error { return nil }); err == nil {
		t.Fatal("same key with new request accepted")
	}
	if _, err = s.Mutate(ctx, r.ID, 1, "fresh-key-0002", map[string]any{"action": "join"}, func(r *Room) error { return nil }); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision accepted: %v", err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	persisted, err := s.Get(ctx, r.ID)
	if err != nil || persisted.Revision != 2 || len(persisted.Members) != 1 {
		t.Fatalf("durability: %+v %v", persisted, err)
	}
}

func TestCompetingWritesHaveOneWinner(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	r, _ := s.Create(ctx, "competing")
	var wg sync.WaitGroup
	wins := make(chan bool, 2)
	for _, k := range []string{"operation-A", "operation-B"} {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, e := s.Mutate(ctx, r.ID, 1, k, k, func(r *Room) error { r.Name = k; return nil })
			if e != nil && !errors.Is(e, ErrConflict) {
				t.Error(e)
			}
			wins <- e == nil
		}(k)
	}
	wg.Wait()
	close(wins)
	n := 0
	for w := range wins {
		if w {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected one winner, got %d", n)
	}
}
