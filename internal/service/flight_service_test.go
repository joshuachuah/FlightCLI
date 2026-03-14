package service

import (
	"context"
	"testing"

	"github.com/xjosh/flightcli/internal/cache"
	"github.com/xjosh/flightcli/internal/models"
)

type stubProvider struct {
	statusCalls int
	searchCalls int
	status      *models.Flight
	search      []models.AirportFlight
}

func (s *stubProvider) GetFlightStatus(ctx context.Context, flightNumber string) (*models.Flight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.statusCalls++
	return s.status, nil
}

func (s *stubProvider) GetAirportFlights(ctx context.Context, airportCode string, flightType string) ([]models.AirportFlight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *stubProvider) SearchFlights(ctx context.Context, from, to string) ([]models.AirportFlight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.searchCalls++
	return s.search, nil
}

func TestGetStatusCachesProviderResponse(t *testing.T) {
	provider := &stubProvider{
		status: &models.Flight{
			FlightNumber: "AA100",
			Airline:      "American Airlines",
			Departure:    "JFK",
			Arrival:      "LAX",
			Status:       "In Flight",
		},
	}
	service := FlightService{
		Provider: provider,
		Cache:    &cache.Cache{Dir: t.TempDir()},
	}

	flight, cached, err := service.GetStatus(context.Background(), "AA100")
	if err != nil {
		t.Fatalf("first GetStatus returned error: %v", err)
	}
	if cached {
		t.Fatalf("expected first GetStatus call to miss cache")
	}
	if provider.statusCalls != 1 {
		t.Fatalf("expected provider to be called once, got %d", provider.statusCalls)
	}
	if flight.FlightNumber != "AA100" {
		t.Fatalf("unexpected flight number %q", flight.FlightNumber)
	}

	flight, cached, err = service.GetStatus(context.Background(), "AA100")
	if err != nil {
		t.Fatalf("second GetStatus returned error: %v", err)
	}
	if !cached {
		t.Fatalf("expected second GetStatus call to hit cache")
	}
	if provider.statusCalls != 1 {
		t.Fatalf("expected provider call count to remain 1, got %d", provider.statusCalls)
	}
	if flight.FlightNumber != "AA100" {
		t.Fatalf("unexpected cached flight number %q", flight.FlightNumber)
	}
}

func TestSearchFlightsCachesProviderResponse(t *testing.T) {
	provider := &stubProvider{
		search: []models.AirportFlight{
			{
				FlightNumber: "DL200",
				Airline:      "Delta Airlines",
				Origin:       "JFK",
				Destination:  "LAX",
				Status:       "Scheduled",
			},
		},
	}
	service := FlightService{
		Provider: provider,
		Cache:    &cache.Cache{Dir: t.TempDir()},
	}

	flights, cached, err := service.SearchFlights(context.Background(), "JFK", "LAX")
	if err != nil {
		t.Fatalf("first SearchFlights returned error: %v", err)
	}
	if cached {
		t.Fatalf("expected first SearchFlights call to miss cache")
	}
	if provider.searchCalls != 1 {
		t.Fatalf("expected provider to be called once, got %d", provider.searchCalls)
	}
	if len(flights) != 1 || flights[0].FlightNumber != "DL200" {
		t.Fatalf("unexpected search result: %#v", flights)
	}

	flights, cached, err = service.SearchFlights(context.Background(), "JFK", "LAX")
	if err != nil {
		t.Fatalf("second SearchFlights returned error: %v", err)
	}
	if !cached {
		t.Fatalf("expected second SearchFlights call to hit cache")
	}
	if provider.searchCalls != 1 {
		t.Fatalf("expected provider call count to remain 1, got %d", provider.searchCalls)
	}
	if len(flights) != 1 || flights[0].FlightNumber != "DL200" {
		t.Fatalf("unexpected cached search result: %#v", flights)
	}
}

func TestGetStatusWithoutCacheHitsProviderEveryTime(t *testing.T) {
	provider := &stubProvider{
		status: &models.Flight{FlightNumber: "AA100"},
	}
	service := FlightService{Provider: provider}

	if _, cached, err := service.GetStatus(context.Background(), "AA100"); err != nil {
		t.Fatalf("first GetStatus returned error: %v", err)
	} else if cached {
		t.Fatalf("expected uncached service to report cached=false")
	}

	if _, cached, err := service.GetStatus(context.Background(), "AA100"); err != nil {
		t.Fatalf("second GetStatus returned error: %v", err)
	} else if cached {
		t.Fatalf("expected uncached service to report cached=false")
	}

	if provider.statusCalls != 2 {
		t.Fatalf("expected provider to be called twice without a cache, got %d", provider.statusCalls)
	}
}
