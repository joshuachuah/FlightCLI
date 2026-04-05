package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

var providerHTTPClientMu sync.Mutex

func withTestHTTPClient(t *testing.T, assert func(*http.Request), handler http.HandlerFunc) {
	t.Helper()
	providerHTTPClientMu.Lock()

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
		providerHTTPClientMu.Unlock()
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

	if _, err := provider.fetchFlights(context.Background(), url.Values{"flight_iata": []string{"AA100&admin=true"}}); err != nil {
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

	flight, err := provider.GetFlightStatus(context.Background(), "KE038")
	if err != nil {
		t.Fatalf("GetFlightStatus returned error: %v", err)
	}
	if flight.FlightNumber != "KE38" {
		t.Fatalf("expected normalized flight number in model, got %q", flight.FlightNumber)
	}
}

func TestFetchFlightsRespectsContextCancellation(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}
	providerHTTPClientMu.Lock()

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}),
	}
	t.Cleanup(func() {
		providerHTTPClient = originalClient
		providerHTTPClientMu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := provider.fetchFlights(ctx, url.Values{"flight_iata": []string{"AA100"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestSearchFlightsIncludesDetailedMetrics(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}

	withTestHTTPClient(t, func(req *http.Request) {
		if got := req.URL.Query().Get("dep_iata"); got != "JFK" {
			t.Fatalf("expected dep_iata=JFK, got %q", got)
		}
		if got := req.URL.Query().Get("arr_iata"); got != "LAX" {
			t.Fatalf("expected arr_iata=LAX, got %q", got)
		}
	}, func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `{"data":[{"flight_status":"active","departure":{"iata":"JFK","timezone":"America/New_York","scheduled":"2026-03-13T10:00:00+00:00","estimated":"2026-03-13T10:15:00+00:00"},"arrival":{"iata":"LAX","timezone":"America/Los_Angeles","scheduled":"2026-03-13T13:00:00+00:00","estimated":"2026-03-13T13:20:00+00:00"},"airline":{"name":"Delta Air Lines"},"flight":{"iata":"DL200"},"live":{"latitude":40.7128,"longitude":-73.9352,"altitude":10515.6,"speed_horizontal":828.9}}]}`)
	})

	flights, err := provider.SearchFlights(context.Background(), "JFK", "LAX")
	if err != nil {
		t.Fatalf("SearchFlights returned error: %v", err)
	}
	if len(flights) != 1 {
		t.Fatalf("expected one flight, got %d", len(flights))
	}

	flight := flights[0]
	if flight.FlightNumber != "DL200" {
		t.Fatalf("unexpected flight number %q", flight.FlightNumber)
	}
	if flight.DepartureTime.IsZero() || flight.ArrivalTime.IsZero() {
		t.Fatalf("expected departure and arrival times to be populated: %#v", flight)
	}
	if flight.Latitude == 0 || flight.Longitude == 0 || flight.Altitude == 0 || flight.Speed == 0 {
		t.Fatalf("expected live telemetry to be populated: %#v", flight)
	}
}

func TestNormalizeFlightWindowRollsArrivalForwardWhenNeeded(t *testing.T) {
	departureLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load departure location: %v", err)
	}
	arrivalLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load arrival location: %v", err)
	}

	departure := time.Date(2026, time.April, 5, 19, 51, 0, 0, departureLoc)
	arrival := time.Date(2026, time.April, 4, 22, 8, 0, 0, arrivalLoc)

	normalizedDeparture, normalizedArrival := normalizeFlightWindow(departure, arrival)
	if !normalizedArrival.After(normalizedDeparture) {
		t.Fatalf("expected normalized arrival to be after departure: departure=%v arrival=%v", normalizedDeparture, normalizedArrival)
	}

	expectedArrival := time.Date(2026, time.April, 5, 22, 8, 0, 0, arrivalLoc)
	if !normalizedArrival.Equal(expectedArrival) {
		t.Fatalf("expected arrival to roll forward to %v, got %v", expectedArrival, normalizedArrival)
	}
}
