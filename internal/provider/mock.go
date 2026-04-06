package provider

import (
	"context"
	"time"

	"github.com/xjosh/flightcli/internal/models"
)

type MockProvider struct{}

func (m *MockProvider) GetFlightStatus(ctx context.Context, flightNumber string) (*models.Flight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func (m *MockProvider) GetAirportFlights(ctx context.Context, airportCode string, flightType string) ([]models.AirportFlight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func (m *MockProvider) SearchFlights(ctx context.Context, from, to string) ([]models.AirportFlight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []models.AirportFlight{
		{
			FlightNumber:  "DL200",
			Airline:       "Delta Airlines",
			Origin:        from,
			Destination:   to,
			Status:        "In Flight",
			Latitude:      40.7128,
			Longitude:     -73.9352,
			Altitude:      34500,
			Speed:         515,
			DepartureTime: time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC),
			ArrivalTime:   time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC),
		},
	}, nil
}
