package models

import "time"

type Flight struct {
	FlightNumber  string
	Airline       string
	Departure     string
	Arrival       string
	Status        string
	Altitude      float64
	Speed         float64
	Latitude      float64
	Longitude     float64
	DepartureTime time.Time
	ArrivalTime   time.Time
}

type AirportFlight struct {
	FlightNumber  string
	Airline       string
	Origin        string
	Destination   string
	Status        string
	ScheduledTime time.Time
}