package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xjosh/flightcli/internal/cache"
	"github.com/xjosh/flightcli/internal/models"
	"github.com/xjosh/flightcli/internal/provider"
)

const (
	flightStatusTTL = 60 * time.Second
	airportTTL      = 5 * time.Minute
	searchTTL       = 5 * time.Minute
)

// FlightService wraps a provider with optional caching.
// Set Cache to nil to disable caching.
type FlightService struct {
	Provider provider.FlightProvider
	Cache    *cache.Cache
}

// GetStatus fetches live flight status, using cache when available.
// Returns (flight, cached, error) - cached is true if the result came from cache.
func (s *FlightService) GetStatus(ctx context.Context, flightNumber string) (*models.Flight, bool, error) {
	flightNumber = strings.TrimSpace(flightNumber)
	if flightNumber == "" {
		return nil, false, fmt.Errorf("flight number is required")
	}

	return getOrFetch(ctx, s.Cache, fmt.Sprintf("status:%s", flightNumber), flightStatusTTL, func(ctx context.Context) (*models.Flight, error) {
		return s.Provider.GetFlightStatus(ctx, flightNumber)
	})
}

// GetAirportFlights fetches airport departure/arrival data, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) GetAirportFlights(ctx context.Context, airportCode, flightType string) ([]models.AirportFlight, bool, error) {
	airportCode = strings.TrimSpace(airportCode)
	flightType = strings.ToLower(strings.TrimSpace(flightType))

	if airportCode == "" {
		return nil, false, fmt.Errorf("airport code is required")
	}
	if flightType == "" {
		return nil, false, fmt.Errorf("flight type is required")
	}

	return getOrFetch(ctx, s.Cache, fmt.Sprintf("airport:%s:%s", airportCode, flightType), airportTTL, func(ctx context.Context) ([]models.AirportFlight, error) {
		return s.Provider.GetAirportFlights(ctx, airportCode, flightType)
	})
}

// SearchFlights searches flights between two airports, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) SearchFlights(ctx context.Context, from, to string) ([]models.AirportFlight, bool, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)

	if from == "" {
		return nil, false, fmt.Errorf("departure airport is required")
	}
	if to == "" {
		return nil, false, fmt.Errorf("arrival airport is required")
	}

	return getOrFetch(ctx, s.Cache, fmt.Sprintf("search:%s:%s", from, to), searchTTL, func(ctx context.Context) ([]models.AirportFlight, error) {
		return s.Provider.SearchFlights(ctx, from, to)
	})
}

func getOrFetch[T any](
	ctx context.Context,
	c *cache.Cache,
	key string,
	ttl time.Duration,
	fetch func(context.Context) (T, error),
) (T, bool, error) {
	var zero T

	if c != nil {
		raw, hit, err := c.Get(key)
		if err == nil && hit {
			var cached T
			if json.Unmarshal(raw, &cached) == nil {
				return cached, true, nil
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return zero, false, err
	}

	value, err := fetch(ctx)
	if err != nil {
		return zero, false, err
	}

	if c != nil {
		_ = c.Set(key, value, ttl)
	}

	return value, false, nil
}
