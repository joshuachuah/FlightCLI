package provider

import "github.com/xjosh/flightcli/internal/models"

type FlightProvider interface {
	GetFlightStatus(flightNumber string) (*models.Flight, error)
	GetAirportFlights(airportCode string, flightType string) ([]models.AirportFlight, error)
}