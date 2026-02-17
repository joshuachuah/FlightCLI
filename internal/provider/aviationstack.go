package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xjosh/flightcli/internal/models"
)

type AviationStackProvider struct {
	APIKey string
}

type aviationStackResponse struct {
	Data []aviationStackFlight `json:"data"`
}

type aviationStackFlight struct {
	FlightStatus string              `json:"flight_status"`
	Departure    aviationStackAirport `json:"departure"`
	Arrival      aviationStackAirport `json:"arrival"`
	Airline      aviationStackAirline `json:"airline"`
	Flight       aviationStackInfo    `json:"flight"`
	Live         *aviationStackLive   `json:"live"`
}

type aviationStackAirport struct {
	Airport   string `json:"airport"`
	IATA      string `json:"iata"`
	Timezone  string `json:"timezone"`
	Scheduled string `json:"scheduled"`
	Estimated string `json:"estimated"`
	Actual    string `json:"actual"`
}

type aviationStackAirline struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type aviationStackInfo struct {
	IATA string `json:"iata"`
}

type aviationStackLive struct {
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Altitude        float64 `json:"altitude"`
	SpeedHorizontal float64 `json:"speed_horizontal"`
	IsGround        bool    `json:"is_ground"`
}

func (a *AviationStackProvider) GetFlightStatus(flightNumber string) (*models.Flight, error) {
	flightIATA := normalizeFlightNumber(strings.ToUpper(strings.TrimSpace(flightNumber)))

	url := fmt.Sprintf("http://api.aviationstack.com/v1/flights?access_key=%s&flight_iata=%s", a.APIKey, flightIATA)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach AviationStack API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AviationStack API returned status %d", resp.StatusCode)
	}

	var data aviationStackResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(data.Data) == 0 {
		return nil, fmt.Errorf("no flight found for %s", flightIATA)
	}

	f := bestFlight(data.Data)

	status := f.FlightStatus
	if status == "scheduled" && f.Live != nil {
		status = "active"
	}

	flight := &models.Flight{
		FlightNumber:  flightIATA,
		Airline:       f.Airline.Name,
		Departure:     f.Departure.IATA,
		Arrival:       f.Arrival.IATA,
		Status:        formatStatus(status),
		DepartureTime: parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled),
		ArrivalTime:   parseLocalTime(f.Arrival.Timezone, f.Arrival.Actual, f.Arrival.Estimated, f.Arrival.Scheduled),
	}

	if f.Live != nil {
		flight.Latitude = f.Live.Latitude
		flight.Longitude = f.Live.Longitude
		flight.Altitude = f.Live.Altitude * 3.28084   // meters to feet
		flight.Speed = f.Live.SpeedHorizontal * 0.621371 // km/h to mph
	}

	return flight, nil
}

func (a *AviationStackProvider) GetAirportFlights(airportCode string, flightType string) ([]models.AirportFlight, error) {
	code := strings.ToUpper(strings.TrimSpace(airportCode))

	param := "dep_iata"
	if flightType == "arrivals" {
		param = "arr_iata"
	}

	url := fmt.Sprintf("http://api.aviationstack.com/v1/flights?access_key=%s&%s=%s", a.APIKey, param, code)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach AviationStack API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AviationStack API returned status %d", resp.StatusCode)
	}

	var data aviationStackResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(data.Data) == 0 {
		return nil, fmt.Errorf("no flights found for airport %s", code)
	}

	var flights []models.AirportFlight
	for _, f := range data.Data {
		scheduled := f.Departure.Scheduled
		tz := f.Departure.Timezone
		if flightType == "arrivals" {
			scheduled = f.Arrival.Scheduled
			tz = f.Arrival.Timezone
		}

		flights = append(flights, models.AirportFlight{
			FlightNumber:  f.Flight.IATA,
			Airline:       f.Airline.Name,
			Origin:        f.Departure.IATA,
			Destination:   f.Arrival.IATA,
			Status:        formatStatus(f.FlightStatus),
			ScheduledTime: parseLocalTime(tz, scheduled),
		})
	}

	return flights, nil
}

// bestFlight picks the most relevant flight from multiple results.
// Prefers active > landed > scheduled, so we don't show a future
// scheduled flight when the current one is still in the air.
func bestFlight(flights []aviationStackFlight) aviationStackFlight {
	priority := map[string]int{"active": 0, "landed": 1, "incident": 2, "diverted": 2, "scheduled": 3, "cancelled": 4}
	best := flights[0]
	bestPri := priority[best.FlightStatus]
	for _, f := range flights[1:] {
		p := priority[f.FlightStatus]
		if p < bestPri {
			best = f
			bestPri = p
		}
	}
	return best
}

// normalizeFlightNumber strips leading zeros from the numeric part.
// AviationStack expects "KE38" not "KE038".
func normalizeFlightNumber(input string) string {
	i := 0
	for i < len(input) && (input[i] >= 'A' && input[i] <= 'Z') {
		i++
	}
	if i == 0 || i >= len(input) {
		return input
	}
	num := strings.TrimLeft(input[i:], "0")
	if num == "" {
		num = "0"
	}
	return input[:i] + num
}

// parseLocalTime re-interprets AviationStack timestamps in the correct timezone.
// AviationStack returns local times with a fake +00:00 offset, so we strip the
// offset and re-parse using the provided IANA timezone (e.g. "America/Chicago").
func parseLocalTime(tz string, values ...string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	for _, v := range values {
		if v == "" {
			continue
		}
		// Parse ignoring the fake offset, then interpret in the real timezone
		t, err := time.Parse("2006-01-02T15:04:05+00:00", v)
		if err != nil {
			continue
		}
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	}
	return time.Time{}
}

func formatStatus(status string) string {
	switch status {
	case "scheduled":
		return "Scheduled"
	case "active":
		return "In Flight"
	case "landed":
		return "Landed"
	case "cancelled":
		return "Cancelled"
	case "incident":
		return "Incident"
	case "diverted":
		return "Diverted"
	default:
		return status
	}
}
