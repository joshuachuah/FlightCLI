package provider

import "github.com/xjosh/flightcli/internal/models"

type MockProvider struct{}

func (m *MockProvider) GetFlightStatus(flightNumber string) (*models.Flight, error) {
	return &models.Flight{
		FlightNumber: flightNumber,
		Airline:      "Delta Airlines",
		Departure:    "JFK",
		Arrival:      "LAX",
		Status:       "In Flight",
		Altitude:     34000,
		Speed:        510,
	}, nil
}

func (m *MockProvider) GetAirportFlights(airportCode string, flightType string) ([]models.AirportFlight, error) {
	return []models.AirportFlight{
		{
			FlightNumber: "DL123",
			Airline:      "Delta Airlines",
			Origin:       "JFK",
			Destination:  "LAX",
			Status:       "Scheduled",
		},
	}, nil
}

func (m *MockProvider) SearchFlights(from, to string) ([]models.AirportFlight, error) {
	return []models.AirportFlight{
		{
			FlightNumber: "DL200",
			Airline:      "Delta Airlines",
			Origin:       from,
			Destination:  to,
			Status:       "Scheduled",
		},
	}, nil
}
