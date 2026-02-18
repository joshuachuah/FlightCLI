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
// Returns (flight, cached, error) — cached is true if the result came from cache.
func (s *FlightService) GetStatus(flightNumber string) (*models.Flight, bool, error) {
	key := fmt.Sprintf("status:%s", flightNumber)

	if s.Cache != nil {
		if raw, hit, _ := s.Cache.Get(key); hit {
			var f models.Flight
			if json.Unmarshal(raw, &f) == nil {
				return &f, true, nil
			}
		}
	}

	f, err := s.Provider.GetFlightStatus(flightNumber)
	if err != nil {
		return nil, false, err
	}

	if s.Cache != nil {
		s.Cache.Set(key, f, flightStatusTTL)
	}

	return f, false, nil
}

// GetAirportFlights fetches airport departure/arrival data, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) GetAirportFlights(airportCode, flightType string) ([]models.AirportFlight, bool, error) {
	key := fmt.Sprintf("airport:%s:%s", airportCode, flightType)

	if s.Cache != nil {
		if raw, hit, _ := s.Cache.Get(key); hit {
			var flights []models.AirportFlight
			if json.Unmarshal(raw, &flights) == nil {
				return flights, true, nil
			}
		}
	}

	flights, err := s.Provider.GetAirportFlights(airportCode, flightType)
	if err != nil {
		return nil, false, err
	}

	if s.Cache != nil {
		s.Cache.Set(key, flights, airportTTL)
	}

	return flights, false, nil
}

// SearchFlights searches flights between two airports, using cache when available.
// Returns (flights, cached, error).
func (s *FlightService) SearchFlights(from, to string) ([]models.AirportFlight, bool, error) {
	key := fmt.Sprintf("search:%s:%s", from, to)

	if s.Cache != nil {
		if raw, hit, _ := s.Cache.Get(key); hit {
			var flights []models.AirportFlight
			if json.Unmarshal(raw, &flights) == nil {
				return flights, true, nil
			}
		}
	}

	flights, err := s.Provider.SearchFlights(from, to)
	if err != nil {
		return nil, false, err
	}

	if s.Cache != nil {
		s.Cache.Set(key, flights, searchTTL)
	}

	return flights, false, nil
}
