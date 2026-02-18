package models

import "time"

type Flight struct {
	FlightNumber  string    `json:"flight_number"`
	Airline       string    `json:"airline"`
	Departure     string    `json:"departure"`
	Arrival       string    `json:"arrival"`
	Status        string    `json:"status"`
	Altitude      float64   `json:"altitude"`
	Speed         float64   `json:"speed"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	DepartureTime time.Time `json:"departure_time,omitempty"`
	ArrivalTime   time.Time `json:"arrival_time,omitempty"`
}

type AirportFlight struct {
	FlightNumber  string    `json:"flight_number"`
	Airline       string    `json:"airline"`
	Origin        string    `json:"origin"`
	Destination   string    `json:"destination"`
	Status        string    `json:"status"`
	ScheduledTime time.Time `json:"scheduled_time,omitempty"`
}
