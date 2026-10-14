package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Watchmode holds the key exclusively on the server. Errors never include URLs,
// response bodies, or credentials. The local cap cannot upgrade the API plan.
type Watchmode struct {
	Key       string
	Client    *http.Client
	Limit     int
	mu        sync.Mutex
	initMu    sync.Mutex
	calls     int
	remaining int
	checked   bool
	stopped   bool
	mapping   map[string]int
}

func (w *Watchmode) get(ctx context.Context, path string, params url.Values, out any) error {
	w.mu.Lock()
	if w.Key == "" {
		w.mu.Unlock()
		return fail("AVAILABILITY_UNKNOWN", 503, "Watchmode key missing")
	}
	if w.Limit == 0 {
		w.Limit = 600
	}
	if w.calls >= w.Limit || w.stopped {
		w.mu.Unlock()
		return fail("QUOTA_EXHAUSTED", 429, "supplier refresh is paused at the local/account quota limit")
	}
	params.Set("apiKey", w.Key)
	req, e := http.NewRequestWithContext(ctx, "GET", "https://api.watchmode.com/v1/"+path+"?"+params.Encode(), nil)
	if e != nil {
		w.mu.Unlock()
		return fail("AVAILABILITY_UNKNOWN", 503, "supplier request failed")
	}
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	w.calls++
	w.mu.Unlock()
	res, e := client.Do(req)
	if e != nil {
		return fail("AVAILABILITY_UNKNOWN", 503, "supplier network request failed")
	}
	defer res.Body.Close()
	if res.StatusCode == 429 {
		w.mu.Lock()
		w.stopped = true
		w.mu.Unlock()
		return fail("QUOTA_EXHAUSTED", 429, "supplier quota exhausted")
	}
	if res.StatusCode != 200 {
		return fail("AVAILABILITY_UNKNOWN", 503, fmt.Sprintf("supplier returned HTTP %d", res.StatusCode))
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(out); e != nil {
		return fail("AVAILABILITY_UNKNOWN", 503, "supplier returned invalid data")
	}
	return nil
}
func (w *Watchmode) Fetch(ctx context.Context, f Film) (Availability, error) {
	w.initMu.Lock()
	w.mu.Lock()
	checked := w.checked
	w.mu.Unlock()
	if !checked {
		var status struct {
			Quota int `json:"quota"`
			Used  int `json:"quotaUsed"`
		}
		if e := w.get(ctx, "status/", url.Values{}, &status); e != nil {
			w.initMu.Unlock()
			return Availability{}, e
		}
		w.mu.Lock()
		w.checked = true
		w.remaining = status.Quota - status.Used
		if status.Quota <= 0 || w.remaining <= 0 {
			w.stopped = true
		}
		w.mu.Unlock()
	}
	w.initMu.Unlock()
	w.mu.Lock()
	remaining := w.remaining
	id := w.mapping[f.IMDb]
	w.mu.Unlock()
	if remaining <= 0 {
		return Availability{}, fail("QUOTA_EXHAUSTED", 429, "remaining supplier quota is unavailable")
	}
	if f.Watchmode > 0 {
		id = f.Watchmode
	}
	if id == 0 {
		if f.IMDb == "" {
			return Availability{}, fail("AVAILABILITY_UNKNOWN", 503, "no verified IMDb mapping")
		}
		var search struct {
			Results []struct {
				ID   int    `json:"id"`
				IMDb string `json:"imdb_id"`
				Type string `json:"type"`
			} `json:"title_results"`
		}
		if e := w.get(ctx, "search/", url.Values{"search_field": {"imdb_id"}, "search_value": {f.IMDb}}, &search); e != nil {
			return Availability{}, e
		}
		for _, r := range search.Results {
			if r.IMDb == f.IMDb && r.Type == "movie" {
				if id != 0 {
					return Availability{}, fail("AVAILABILITY_UNKNOWN", 503, "ambiguous supplier identifier")
				}
				id = r.ID
			}
		}
		if id == 0 {
			return Availability{}, fail("AVAILABILITY_UNKNOWN", 503, "supplier identifier not found")
		}
		w.mu.Lock()
		if w.mapping == nil {
			w.mapping = map[string]int{}
		}
		w.mapping[f.IMDb] = id
		w.remaining--
		w.mu.Unlock()
	}
	var rows []struct {
		Name   string `json:"name"`
		ID     int    `json:"source_id"`
		Type   string `json:"type"`
		Region string `json:"region"`
		URL    string `json:"web_url"`
	}
	if e := w.get(ctx, "title/"+strconv.Itoa(id)+"/sources/", url.Values{"regions": {"US"}}, &rows); e != nil {
		return Availability{}, e
	}
	now := time.Now().UTC()
	av := Availability{FilmID: f.ID, Known: true, FetchedAt: now, ExpiresAt: now.Add(24 * time.Hour), Offers: []Offer{}}
	for _, r := range rows {
		if r.Region == "US" {
			offer := Offer{Name: r.Name, Region: r.Region, ProviderID: r.ID, Monetization: r.Type, URL: r.URL}
			found := false
			for _, v := range av.Offers {
				if hashRequest(v) == hashRequest(offer) {
					found = true
				}
			}
			if !found {
				av.Offers = append(av.Offers, offer)
			}
		}
	}
	w.mu.Lock()
	w.remaining--
	w.mu.Unlock()
	return av, nil
}
func (w *Watchmode) Status() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()
	return map[string]any{"configured": w.Key != "", "quota_checked": w.checked, "remaining_at_last_check_minus_local_calls": w.remaining, "process_requests": w.calls, "process_request_cap": w.Limit, "paused": w.stopped}
}
