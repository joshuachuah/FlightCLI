package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/xjosh/flightcli/internal/airlines"
	"github.com/xjosh/flightcli/internal/models"
	"github.com/xjosh/flightcli/internal/sanitize"
)

const aviationStackEndpoint = "https://api.aviationstack.com/v1/flights"

var accessKeyQueryPattern = regexp.MustCompile(`(access_key=)[^&\s"]*`)

const (
	metersToFeet = 3.28084
	kmhToMph     = 0.621371
)

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
	ICAO string `json:"icao"`
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
	normalizedFlightNumber := normalizeFlightNumber(strings.ToUpper(strings.TrimSpace(flightNumber)))
	queries := flightNumberQueries(normalizedFlightNumber)
	notFoundErr := fmt.Errorf("no flight found for %s", normalizedFlightNumber)

	var data []aviationStackFlight
	for _, query := range queries {
		flights, err := a.fetchFlights(ctx, query.params)
		if err != nil {
			return nil, err
		}
		if len(flights) > 0 {
			data = flights
			break
		}
	}
	if len(data) == 0 {
		return nil, notFoundErr
	}

	f := bestFlight(data)
	status := effectiveFlightStatus(f)

	departureTime, arrivalTime := normalizeFlightWindow(
		parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled),
		parseLocalTime(f.Arrival.Timezone, f.Arrival.Actual, f.Arrival.Estimated, f.Arrival.Scheduled),
	)

	// Prefer the IATA flight number from the API response (e.g. "UA2189")
	// over the user's input (which may be ICAO like "UAL2189").
	displayNumber := f.Flight.IATA
	if displayNumber == "" {
		displayNumber = normalizedFlightNumber
	}

	flight := &models.Flight{
		FlightNumber:  displayNumber,
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
		flight.Altitude = f.Live.Altitude * metersToFeet
		flight.Speed = f.Live.SpeedHorizontal * kmhToMph
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
		return nil, redactedErrorf(err, "building AviationStack request for %s", redactAccessKey(endpoint.String()))
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return nil, redactedErrorf(err, "failed to reach AviationStack API: %s", sanitizedProviderErrorText(err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AviationStack API returned status %d", resp.StatusCode)
	}

	var data aviationStackResponse
	body := io.LimitReader(resp.Body, 10<<20)
	if err := json.NewDecoder(body).Decode(&data); err != nil {
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
		flight.Altitude = f.Live.Altitude * metersToFeet
		flight.Speed = f.Live.SpeedHorizontal * kmhToMph
	}

	return flight
}

// bestFlight picks the most relevant flight from multiple results.
// It prefers status priority (active > landed > scheduled), but also
// considers recency: a stale completed flight from over a day before
// a scheduled one should not win just because "landed" outranks
// "scheduled". Specifically, a higher-priority flight is skipped if
// it departed more than 24 hours before a lower-priority alternative.
func bestFlight(flights []aviationStackFlight) aviationStackFlight {
	best := flights[0]
	bestPri := flightPriority(effectiveFlightStatus(best))
	bestDistance := departureDistanceFromNow(best)
	bestDeparture := departureTimeOf(best)
	for _, f := range flights[1:] {
		p := flightPriority(effectiveFlightStatus(f))
		distance := departureDistanceFromNow(f)
		depart := departureTimeOf(f)

		if p < bestPri {
			// New flight has better priority. But if it departed more than
			// 24 hours before the current best, it's stale — skip it.
			if !bestDeparture.IsZero() && !depart.IsZero() && bestDeparture.Sub(depart) > 24*time.Hour {
				continue
			}
			best = f
			bestPri = p
			bestDistance = distance
			bestDeparture = depart
		} else if p > bestPri {
			// Current best has better priority. But if it departed more than
			// 24 hours before this more recent flight, it's stale — replace.
			if !bestDeparture.IsZero() && !depart.IsZero() && depart.Sub(bestDeparture) > 24*time.Hour {
				best = f
				bestPri = p
				bestDistance = distance
				bestDeparture = depart
			}
		} else if distance < bestDistance {
			// Same priority: prefer closer to now
			best = f
			bestPri = p
			bestDistance = distance
			bestDeparture = depart
		}
	}
	return best
}

func departureTimeOf(f aviationStackFlight) time.Time {
	return parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled)
}

func departureDistanceFromNow(f aviationStackFlight) time.Duration {
	departure := parseLocalTime(f.Departure.Timezone, f.Departure.Actual, f.Departure.Estimated, f.Departure.Scheduled)
	if departure.IsZero() {
		return 1 << 62
	}

	distance := time.Since(departure)
	if distance < 0 {
		return -distance
	}
	return distance
}

func effectiveFlightStatus(f aviationStackFlight) string {
	status := strings.ToLower(strings.TrimSpace(f.FlightStatus))

	// AviationStack marks flights as "active" near scheduled departure even
	// if the plane is still at the gate. Downgrade to "scheduled" unless we
	// have evidence the flight has actually departed:
	//   - Live tracking with the plane off the ground (IsGround == false), OR
	//   - An actual departure time has been recorded
	//
	// If the only evidence is ground telemetry (IsGround == true) with no
	// actual departure, the plane hasn't taken off — treat as "scheduled".
	if status == "active" {
		genuinelyAirborne := (f.Live != nil && !f.Live.IsGround) || f.Departure.Actual != ""
		if !genuinelyAirborne {
			return "scheduled"
		}
	}

	if status == "scheduled" && f.Live != nil && !f.Live.IsGround {
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

func normalizeFlightNumber(input string) string {
	if len(input) >= 3 && input[0] >= '0' && input[0] <= '9' && input[1] >= 'A' && input[1] <= 'Z' {
		allDigits := true
		for i := 2; i < len(input); i++ {
			if input[i] < '0' || input[i] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			num := strings.TrimLeft(input[2:], "0")
			if num == "" {
				num = "0"
			}
			return input[:2] + num
		}
	}

	i := 0
	for i < len(input) && (input[i] >= 'A' && input[i] <= 'Z') {
		i++
	}
	if i == 0 || i >= len(input) {
		return input
	}

	prefix := input[:i]
	num := strings.TrimLeft(input[i:], "0")
	if num == "" {
		num = "0"
	}

	return prefix + num
}

type flightQuery struct {
	params url.Values
}

func flightNumberQueries(input string) []flightQuery {
	input = normalizeFlightNumber(strings.ToUpper(strings.TrimSpace(input)))

	i := 0
	for i < len(input) && (input[i] >= 'A' && input[i] <= 'Z') {
		i++
	}
	if i == 3 {
		prefix := input[:i]
		if i < len(input) {
			num := input[i:]
			queries := []flightQuery{
				{
					params: url.Values{"flight_icao": []string{input}},
				},
			}

			// Fall back to the IATA flight number if we know the IATA code.
			// The AviationStack free tier may not support flight_icao, so this
			// ensures we can still find flights like UAL2189 -> UA2189.
			if iata := airlines.IATACode(prefix); iata != "" {
				iataFlight := iata + num
				queries = append(queries, flightQuery{
					params: url.Values{"flight_iata": []string{iataFlight}},
				})
			}

			return queries
		}
	}

	return []flightQuery{
		{
			params: url.Values{"flight_iata": []string{input}},
		},
	}
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

type redactedError struct {
	message string
	err     error
}

func (e redactedError) Error() string {
	return e.message
}

func (e redactedError) Unwrap() error {
	return e.err
}

func redactedErrorf(err error, format string, args ...interface{}) error {
	return redactedError{
		message: fmt.Sprintf(format, args...),
		err:     err,
	}
}

func redactAccessKey(rawURL string) string {
	return redactAccessKeyInText(rawURL)
}

func redactAccessKeyInText(s string) string {
	return accessKeyQueryPattern.ReplaceAllString(s, "${1}[REDACTED]")
}

func sanitizedProviderErrorText(s string) string {
	return sanitize.TerminalString(redactAccessKeyInText(s))
}
