package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xjosh/flightcli/internal/models"
)

const aviationStackEndpoint = "https://api.aviationstack.com/v1/flights"

type AviationStackProvider struct {
	APIKey string
}

type aviationStackResponse struct {
	Data []aviationStackFlight `json:"data"`
}

type aviationStackFlight struct {
	FlightStatus string               `json:"flight_status"`
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

func (a *AviationStackProvider) GetFlightStatus(ctx context.Context, flightNumber string) (*models.Flight, error) {
	flightIATA := normalizeFlightNumber(strings.ToUpper(strings.TrimSpace(flightNumber)))
	data, err := a.fetchFlights(ctx, url.Values{"flight_iata": []string{flightIATA}})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no flight found for %s", flightIATA)
	}

	f := bestFlight(data)
	status := effectiveFlightStatus(f)

	departureTime, arrivalTime := normalizeFlightWindow(
		parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled),
		parseLocalTime(f.Arrival.Timezone, f.Arrival.Actual, f.Arrival.Estimated, f.Arrival.Scheduled),
	)

	flight := &models.Flight{
		FlightNumber:  flightIATA,
		Airline:       f.Airline.Name,
		Departure:     f.Departure.IATA,
		Arrival:       f.Arrival.IATA,
		Status:        formatStatus(status),
		DepartureTime: departureTime,
		ArrivalTime:   arrivalTime,
	}

	if f.Live != nil {
		flight.Latitude = f.Live.Latitude
		flight.Longitude = f.Live.Longitude
		flight.Altitude = f.Live.Altitude * 3.28084
		flight.Speed = f.Live.SpeedHorizontal * 0.621371
	}

	return flight, nil
}

func (a *AviationStackProvider) GetAirportFlights(ctx context.Context, airportCode string, flightType string) ([]models.AirportFlight, error) {
	code := strings.ToUpper(strings.TrimSpace(airportCode))
	flightType = strings.ToLower(strings.TrimSpace(flightType))

	param := "dep_iata"
	if flightType == "arrivals" {
		param = "arr_iata"
	} else if flightType != "departures" {
		return nil, fmt.Errorf("invalid flight type %q: must be 'departures' or 'arrivals'", flightType)
	}

	data, err := a.fetchFlights(ctx, url.Values{param: []string{code}})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no flights found for airport %s", code)
	}

	flights := make([]models.AirportFlight, 0, len(data))
	for _, f := range data {
		scheduled := f.Departure.Scheduled
		tz := f.Departure.Timezone
		if flightType == "arrivals" {
			scheduled = f.Arrival.Scheduled
			tz = f.Arrival.Timezone
		}
		flights = append(flights, airportFlightFromAviationStack(f, parseLocalTime(tz, scheduled)))
	}

	return flights, nil
}

func (a *AviationStackProvider) SearchFlights(ctx context.Context, from, to string) ([]models.AirportFlight, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))

	data, err := a.fetchFlights(ctx, url.Values{
		"dep_iata": []string{from},
		"arr_iata": []string{to},
	})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no flights found for route %s -> %s", from, to)
	}

	flights := make([]models.AirportFlight, 0, len(data))
	for _, f := range data {
		flights = append(flights, airportFlightFromAviationStack(f, parseLocalTime(f.Departure.Timezone, f.Departure.Scheduled)))
	}
	return flights, nil
}

func (a *AviationStackProvider) fetchFlights(ctx context.Context, params url.Values) ([]aviationStackFlight, error) {
	endpoint, err := url.Parse(aviationStackEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid AviationStack endpoint: %w", err)
	}

	query := endpoint.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	query.Set("access_key", a.APIKey)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building AviationStack request: %w", err)
	}
	resp, err := providerHTTPClient.Do(req)
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

	return data.Data, nil
}

func airportFlightFromAviationStack(f aviationStackFlight, scheduled time.Time) models.AirportFlight {
	status := effectiveFlightStatus(f)

	departureTime, arrivalTime := normalizeFlightWindow(
		parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled),
		parseLocalTime(f.Arrival.Timezone, f.Arrival.Actual, f.Arrival.Estimated, f.Arrival.Scheduled),
	)

	flight := models.AirportFlight{
		FlightNumber:  f.Flight.IATA,
		Airline:       f.Airline.Name,
		Origin:        f.Departure.IATA,
		Destination:   f.Arrival.IATA,
		Status:        formatStatus(status),
		DepartureTime: departureTime,
		ArrivalTime:   arrivalTime,
		ScheduledTime: scheduled,
	}

	if f.Live != nil {
		flight.Latitude = f.Live.Latitude
		flight.Longitude = f.Live.Longitude
		flight.Altitude = f.Live.Altitude * 3.28084
		flight.Speed = f.Live.SpeedHorizontal * 0.621371
	}

	return flight
}

// bestFlight picks the most relevant flight from multiple results.
// Prefers active > landed > scheduled, so we don't show a future
// scheduled flight when the current one is still in the air.
func bestFlight(flights []aviationStackFlight) aviationStackFlight {
	best := flights[0]
	bestPri := flightPriority(effectiveFlightStatus(best))
	for _, f := range flights[1:] {
		p := flightPriority(effectiveFlightStatus(f))
		if p < bestPri {
			best = f
			bestPri = p
		}
	}
	return best
}

func effectiveFlightStatus(f aviationStackFlight) string {
	status := strings.ToLower(strings.TrimSpace(f.FlightStatus))
	if status == "scheduled" && f.Live != nil {
		return "active"
	}
	return status
}

func flightPriority(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 0
	case "landed":
		return 1
	case "incident", "diverted":
		return 2
	case "scheduled":
		return 3
	case "cancelled":
		return 4
	default:
		return 99
	}
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
		t, err := time.Parse("2006-01-02T15:04:05+00:00", v)
		if err != nil {
			continue
		}
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	}
	return time.Time{}
}

func normalizeFlightWindow(departure, arrival time.Time) (time.Time, time.Time) {
	if departure.IsZero() || arrival.IsZero() || arrival.After(departure) {
		return departure, arrival
	}

	for i := 0; i < 3 && !arrival.After(departure); i++ {
		arrival = arrival.AddDate(0, 0, 1)
	}

	return departure, arrival
}

func formatStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
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
