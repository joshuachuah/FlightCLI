package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withTestHTTPClient(t *testing.T, assert func(*http.Request), handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert(req)
			rewritten := req.Clone(req.Context())
			rewritten.URL.Scheme = serverURL.Scheme
			rewritten.URL.Host = serverURL.Host
			return http.DefaultTransport.RoundTrip(rewritten)
		}),
	}
	t.Cleanup(func() {
		providerHTTPClient = originalClient
	})
}

func TestFetchFlightsUsesHTTPSAndEncodesQuery(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}

	withTestHTTPClient(t, func(req *http.Request) {
		if req.URL.Scheme != "https" {
			t.Fatalf("expected https scheme, got %q", req.URL.Scheme)
		}
		if got := req.URL.Query().Get("access_key"); got != "secret-key" {
			t.Fatalf("expected access key to be propagated, got %q", got)
		}
		if got := req.URL.Query().Get("flight_iata"); got != "AA100&admin=true" {
			t.Fatalf("expected encoded flight number, got %q", got)
		}
		if got := req.URL.Query().Get("admin"); got != "" {
			t.Fatalf("expected injected query param to be absent, got %q", got)
		}
	}, func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `{"data":[]}`)
	})

	if _, err := provider.fetchFlights(url.Values{"flight_iata": []string{"AA100&admin=true"}}); err != nil {
		t.Fatalf("fetchFlights returned error: %v", err)
	}
}

func TestGetFlightStatusNormalizesFlightNumber(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}

	withTestHTTPClient(t, func(req *http.Request) {
		if got := req.URL.Query().Get("flight_iata"); got != "KE38" {
			t.Fatalf("expected normalized flight number KE38, got %q", got)
		}
	}, func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `{"data":[{"flight_status":"active","departure":{"iata":"ICN","timezone":"Asia/Seoul","scheduled":"2026-03-13T10:00:00+00:00"},"arrival":{"iata":"JFK","timezone":"America/New_York","scheduled":"2026-03-13T20:00:00+00:00"},"airline":{"name":"Korean Air"},"flight":{"iata":"KE38"}}]}`)
	})

	flight, err := provider.GetFlightStatus("KE038")
	if err != nil {
		t.Fatalf("GetFlightStatus returned error: %v", err)
	}
	if flight.FlightNumber != "KE38" {
		t.Fatalf("expected normalized flight number in model, got %q", flight.FlightNumber)
	}
}
