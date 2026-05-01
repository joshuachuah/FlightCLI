package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert(req)
			recorder := httptest.NewRecorder()
			handler(recorder, req)
			return recorder.Result(), nil
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

func TestGetFlightStatusConvertsICAOToIATA(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}

	withTestHTTPClient(t, func(req *http.Request) {
		if got := req.URL.Query().Get("flight_iata"); got != "UA2189" {
			t.Fatalf("expected IATA flight number UA2189, got %q", got)
		}
	}, func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `{"data":[{"flight_status":"scheduled","departure":{"iata":"EWR","timezone":"America/New_York","scheduled":"2026-03-13T08:00:00+00:00"},"arrival":{"iata":"SFO","timezone":"America/Los_Angeles","scheduled":"2026-03-13T11:00:00+00:00"},"airline":{"name":"United Airlines"},"flight":{"iata":"UA2189"}}]}`)
	})

	flight, err := provider.GetFlightStatus(context.Background(), "UAL2189")
	if err != nil {
		t.Fatalf("GetFlightStatus returned error: %v", err)
	}
	if flight.FlightNumber != "UA2189" {
		t.Fatalf("expected IATA flight number UA2189, got %q", flight.FlightNumber)
	}
}

func TestNormalizeFlightNumberPreservesIATA(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		// IATA codes are 2 chars — should pass through unchanged
		{"UA2189", "UA2189"},
		{"AA100", "AA100"},
		{"DL502", "DL502"},
		// ICAO codes are 3 chars — should be converted
		{"UAL2189", "UA2189"},
		{"AAL100", "AA100"},
		{"DAL502", "DL502"},
		{"BAW117", "BA117"},
		{"ACA901", "AC901"},  // Air Canada (ICAO ACA, IATA AC)
		{"THY777", "TK777"},  // Turkish Airlines (ICAO THY, IATA TK)
		{"TSC200", "TS200"},  // Air Transat (ICAO TSC, IATA TS)
		// Leading zeros stripped regardless
		{"KE038", "KE38"},
		// 4-char prefix doesn't match any ICAO code — left alone
		{"TEST1", "TEST1"},
		// Edge cases
		{"", ""},           // empty string
		{"1234", "1234"},   // all digits, no prefix
	}
	for _, tt := range tests {
		got := normalizeFlightNumber(tt.input)
		if got != tt.want {
			t.Errorf("normalizeFlightNumber(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIcaoToIATAKeysAreThreeLetters(t *testing.T) {
	for icao, iata := range icaoToIATA {
		if len(icao) != 3 {
			t.Errorf("icaoToIATA key %q has length %d, want 3", icao, len(icao))
		}
		for _, c := range icao {
			if c < 'A' || c > 'Z' {
				t.Errorf("icaoToIATA key %q contains non-A-Z rune %q", icao, string(c))
			}
		}
		if len(iata) < 1 || len(iata) > 2 {
			t.Errorf("icaoToIATA[%q] = %q has length %d, want 1-2", icao, iata, len(iata))
		}
	}
}

func TestGetFlightStatusPrefersScheduledFlightWithLiveTelemetry(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}

	withTestHTTPClient(t, func(req *http.Request) {
		if got := req.URL.Query().Get("flight_iata"); got != "AA100" {
			t.Fatalf("expected flight_iata=AA100, got %q", got)
		}
	}, func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"flight_status":"landed","departure":{"iata":"JFK","timezone":"America/New_York","scheduled":"2026-03-13T08:00:00+00:00"},"arrival":{"iata":"LAX","timezone":"America/Los_Angeles","scheduled":"2026-03-13T11:00:00+00:00"},"airline":{"name":"Old Air"},"flight":{"iata":"AA100"}},
			{"flight_status":"scheduled","departure":{"iata":"JFK","timezone":"America/New_York","scheduled":"2026-03-13T09:00:00+00:00"},"arrival":{"iata":"LAX","timezone":"America/Los_Angeles","scheduled":"2026-03-13T12:00:00+00:00"},"airline":{"name":"Live Air"},"flight":{"iata":"AA100"},"live":{"latitude":40.7128,"longitude":-73.9352,"altitude":10000,"speed_horizontal":800}}
		]}`)
	})

	flight, err := provider.GetFlightStatus(context.Background(), "AA100")
	if err != nil {
		t.Fatalf("GetFlightStatus returned error: %v", err)
	}
	if flight.Status != "In Flight" {
		t.Fatalf("expected live scheduled flight to be selected as In Flight, got %q", flight.Status)
	}
	if flight.Airline != "Live Air" {
		t.Fatalf("expected live scheduled flight to be selected, got airline %q", flight.Airline)
	}
	if flight.Latitude == 0 || flight.Longitude == 0 {
		t.Fatalf("expected telemetry from selected live flight, got %#v", flight)
	}
}

func TestBestFlightPrefersDepartureClosestToNowWhenPriorityTies(t *testing.T) {
	now := time.Now().UTC()
	oldDeparture := now.Add(-24 * time.Hour).Format("2006-01-02T15:04:05+00:00")
	currentDeparture := now.Add(15 * time.Minute).Format("2006-01-02T15:04:05+00:00")

	best := bestFlight([]aviationStackFlight{
		{
			FlightStatus: "scheduled",
			Departure: aviationStackAirport{
				IATA:      "KUL",
				Timezone:  "UTC",
				Scheduled: oldDeparture,
			},
			Airline: aviationStackAirline{Name: "Old Air"},
			Flight:  aviationStackInfo{IATA: "D7504"},
		},
		{
			FlightStatus: "scheduled",
			Departure: aviationStackAirport{
				IATA:      "KUL",
				Timezone:  "UTC",
				Scheduled: currentDeparture,
			},
			Airline: aviationStackAirline{Name: "Current Air"},
			Flight:  aviationStackInfo{IATA: "D7504"},
		},
	})

	if best.Airline.Name != "Current Air" {
		t.Fatalf("expected closest same-priority flight to be selected, got %q", best.Airline.Name)
	}
}

func TestDepartureDistanceFromNowUsesActualThenEstimatedThenScheduled(t *testing.T) {
	now := time.Now().UTC()
	actual := now.Add(5 * time.Minute).Format("2006-01-02T15:04:05+00:00")
	estimated := now.Add(2 * time.Hour).Format("2006-01-02T15:04:05+00:00")
	scheduled := now.Add(24 * time.Hour).Format("2006-01-02T15:04:05+00:00")

	distance := departureDistanceFromNow(aviationStackFlight{
		Departure: aviationStackAirport{
			Timezone:  "UTC",
			Scheduled: scheduled,
			Estimated: estimated,
			Actual:    actual,
		},
	})

	if distance > 10*time.Minute {
		t.Fatalf("expected distance to use actual departure first, got %s", distance)
	}
}

func TestDepartureDistanceFromNowTreatsMissingTimeAsFarAway(t *testing.T) {
	distance := departureDistanceFromNow(aviationStackFlight{})

	if distance != 1<<62 {
		t.Fatalf("expected missing departure time to be max distance, got %s", distance)
	}
}

func TestEffectiveFlightStatusDowngradesActiveWithoutDepartureEvidence(t *testing.T) {
	status := effectiveFlightStatus(aviationStackFlight{
		FlightStatus: "active",
		Departure: aviationStackAirport{
			Scheduled: "2026-03-13T09:00:00+00:00",
		},
	})

	if status != "scheduled" {
		t.Fatalf("expected active flight without live or actual departure evidence to be scheduled, got %q", status)
	}
}

func TestEffectiveFlightStatusTrustsActiveWithAirborneLiveOrActualDeparture(t *testing.T) {
	// Live data with IsGround == false means the plane is in the air
	withAirborne := effectiveFlightStatus(aviationStackFlight{
		FlightStatus: "active",
		Live:         &aviationStackLive{IsGround: false},
	})
	if withAirborne != "active" {
		t.Fatalf("expected active flight with airborne live data to stay active, got %q", withAirborne)
	}

	// Actual departure time recorded means the plane has left
	withActual := effectiveFlightStatus(aviationStackFlight{
		FlightStatus: "active",
		Departure: aviationStackAirport{
			Actual: "2026-03-13T09:05:00+00:00",
		},
	})
	if withActual != "active" {
		t.Fatalf("expected active flight with actual departure to stay active, got %q", withActual)
	}
}

func TestEffectiveFlightStatusDowngradesActiveWithOnlyGroundTelemetry(t *testing.T) {
	// Live data with IsGround == true and no actual departure = still at gate
	withGround := effectiveFlightStatus(aviationStackFlight{
		FlightStatus: "active",
		Live:         &aviationStackLive{IsGround: true},
	})
	if withGround != "scheduled" {
		t.Fatalf("expected active flight with only ground telemetry to be downgraded, got %q", withGround)
	}
}

func TestBestFlightPrefersCurrentScheduledOverStaleLanded(t *testing.T) {
	now := time.Now().UTC()
	staleDeparture := now.Add(-48 * time.Hour).Format("2006-01-02T15:04:05+00:00")
	currentDeparture := now.Add(15 * time.Minute).Format("2006-01-02T15:04:05+00:00")

	best := bestFlight([]aviationStackFlight{
		{
			FlightStatus: "landed",
			Departure: aviationStackAirport{
				IATA:      "KUL",
				Timezone:  "UTC",
				Scheduled: staleDeparture,
			},
			Airline: aviationStackAirline{Name: "Old Landed Air"},
			Flight:  aviationStackInfo{IATA: "D7504"},
		},
		{
			FlightStatus: "scheduled",
			Departure: aviationStackAirport{
				IATA:      "ICN",
				Timezone:  "UTC",
				Scheduled: currentDeparture,
			},
			Airline: aviationStackAirline{Name: "Current Air"},
			Flight:  aviationStackInfo{IATA: "D7504"},
		},
	})

	if best.Airline.Name != "Current Air" {
		t.Fatalf("expected current scheduled flight to beat stale landed flight, got %q", best.Airline.Name)
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

func TestFetchFlightsRedactsAccessKeyFromTransportError(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}
	providerHTTPClientMu.Lock()

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp failed")
		}),
	}
	t.Cleanup(func() {
		providerHTTPClient = originalClient
		providerHTTPClientMu.Unlock()
	})

	_, err := provider.fetchFlights(context.Background(), url.Values{"flight_iata": []string{"AA100"}})
	if err == nil {
		t.Fatal("expected transport error")
	}

	message := err.Error()
	if strings.Contains(message, "secret-key") {
		t.Fatalf("transport error leaked API key: %q", message)
	}
	if !strings.Contains(message, "access_key=[REDACTED]") {
		t.Fatalf("transport error did not include redacted access key: %q", message)
	}
}

func TestFetchFlightsSanitizesTransportErrorControls(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}
	providerHTTPClientMu.Lock()

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed\x1b]0;spoof\a")
		}),
	}
	t.Cleanup(func() {
		providerHTTPClient = originalClient
		providerHTTPClientMu.Unlock()
	})

	_, err := provider.fetchFlights(context.Background(), url.Values{"flight_iata": []string{"AA100"}})
	if err == nil {
		t.Fatal("expected transport error")
	}

	message := err.Error()
	for _, forbidden := range []string{"\x1b", "\a", "spoof", "secret-key"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("transport error %q still contains forbidden content %q", message, forbidden)
		}
	}
}

func TestFetchFlightsRedactedTransportErrorStillUnwraps(t *testing.T) {
	provider := &AviationStackProvider{APIKey: "secret-key"}
	providerHTTPClientMu.Lock()

	originalClient := providerHTTPClient
	providerHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, context.Canceled
		}),
	}
	t.Cleanup(func() {
		providerHTTPClient = originalClient
		providerHTTPClientMu.Unlock()
	})

	_, err := provider.fetchFlights(context.Background(), url.Values{"flight_iata": []string{"AA100"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected redacted error to unwrap context cancellation, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("redacted cancellation error leaked API key: %q", err.Error())
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

func TestNormalizeFlightWindowPreservesWallClockAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	departure := time.Date(2026, time.March, 8, 21, 0, 0, 0, loc)
	arrival := time.Date(2026, time.March, 7, 22, 8, 0, 0, loc)

	_, normalizedArrival := normalizeFlightWindow(departure, arrival)
	expectedArrival := time.Date(2026, time.March, 8, 22, 8, 0, 0, loc)
	if !normalizedArrival.Equal(expectedArrival) {
		t.Fatalf("expected DST-normalized arrival to be %v, got %v", expectedArrival, normalizedArrival)
	}
}
