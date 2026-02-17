package provider

import "github.com/xjosh/flightcli/internal/models"

type MockProvider struct{}

func (m *MockProvider) GetFlightStatus(flightNumber string) (*models.Flight, error) {
	return &models.Flight{
		FlightNumber: flightNumber,
		Airline:      "Delta Airlines",
		Departure:    "JFK",
		Arrival:      "LAX",
		Status:       "En Route",
		Altitude:     34000,
		Speed:        510,
	}, nil
}
