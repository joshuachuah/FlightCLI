package service

import (
	"encoding/json"
	"fmt"
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
func (s *FlightService) GetStatus(flightNumber string) (*models.Flight, bool, error) {
	return getOrFetch(s.Cache, fmt.Sprintf("status:%s", flightNumber), flightStatusTTL, func() (*models.Flight, error) {
		return s.Provider.GetFlightStatus(flightNumber)
	})
}

// GetAirportFlights fetches airport departure/arrival data, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) GetAirportFlights(airportCode, flightType string) ([]models.AirportFlight, bool, error) {
	return getOrFetch(s.Cache, fmt.Sprintf("airport:%s:%s", airportCode, flightType), airportTTL, func() ([]models.AirportFlight, error) {
		return s.Provider.GetAirportFlights(airportCode, flightType)
	})
}

// SearchFlights searches flights between two airports, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) SearchFlights(from, to string) ([]models.AirportFlight, bool, error) {
	return getOrFetch(s.Cache, fmt.Sprintf("search:%s:%s", from, to), searchTTL, func() ([]models.AirportFlight, error) {
		return s.Provider.SearchFlights(from, to)
	})
}

func getOrFetch[T any](c *cache.Cache, key string, ttl time.Duration, fetch func() (T, error)) (T, bool, error) {
	var zero T

	if c != nil {
		if raw, hit, _ := c.Get(key); hit {
			var cached T
			if json.Unmarshal(raw, &cached) == nil {
				return cached, true, nil
			}
		}
	}

	value, err := fetch()
	if err != nil {
		return zero, false, err
	}

	if c != nil {
		c.Set(key, value, ttl)
	}

	return value, false, nil
}
