package workflow_test

import (
	"commonreel/internal/workflow"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type supplierTransport func(*http.Request) (*http.Response, error)

func (f supplierTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWatchmodeDeduplicatesExactOffersAndPreservesPurchaseTypes(t *testing.T) {
	client := &http.Client{Transport: supplierTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"quota":100,"quotaUsed":0}`
		if r.URL.Path == "/v1/title/42/sources/" {
			body = `[
				{"name":"Example","source_id":8,"type":"sub","region":"US","web_url":"https://example.com/sub"},
				{"name":"Example","source_id":8,"type":"sub","region":"US","web_url":"https://example.com/sub"},
				{"name":"Example","source_id":8,"type":"rent","region":"US","web_url":"https://example.com/rent"},
				{"name":"Example","source_id":8,"type":"sub","region":"CA","web_url":"https://example.com/ca"}
			]`
		} else if r.URL.Path != "/v1/status/" {
			t.Errorf("unexpected supplier request: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	w := &workflow.Watchmode{Key: "controlled-test-key", Client: client}
	availability, err := w.Fetch(context.Background(), workflow.Film{ID: "a", Watchmode: 42})
	if err != nil {
		t.Fatal(err)
	}
	want := []workflow.Offer{
		{Name: "Example", Region: "US", ProviderID: 8, Monetization: "sub", URL: "https://example.com/sub"},
		{Name: "Example", Region: "US", ProviderID: 8, Monetization: "rent", URL: "https://example.com/rent"},
	}
	if !availability.Known || len(availability.Offers) != len(want) {
		t.Fatal(availability)
	}
	for i := range want {
		if availability.Offers[i] != want[i] {
			t.Fatal("offer type or order changed", availability.Offers)
		}
	}
}
