package provider

import (
	"context"

	"github.com/xjosh/flightcli/internal/models"
)

type FlightProvider interface {
	GetFlightStatus(ctx context.Context, flightNumber string) (*models.Flight, error)
	GetAirportFlights(ctx context.Context, airportCode string, flightType string) ([]models.AirportFlight, error)
	SearchFlights(ctx context.Context, from, to string) ([]models.AirportFlight, error)
}
