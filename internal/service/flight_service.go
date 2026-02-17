package service

import (
	"github.com/xjosh/flightcli/internal/models"
	"github.com/xjosh/flightcli/internal/provider"
)

type FlightService struct {
	Provider provider.FlightProvider
}

func (s *FlightService) GetStatus(flightNumber string) (*models.Flight, error) {
	return s.Provider.GetFlightStatus(flightNumber)
}

func (s *FlightService) GetAirportFlights(airportCode string, flightType string) ([]models.AirportFlight, error) {
	return s.Provider.GetAirportFlights(airportCode, flightType)
}