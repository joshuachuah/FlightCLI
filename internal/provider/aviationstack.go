package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	Airport string `json:"airport"`
	IATA    string `json:"iata"`
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

	f := data.Data[0]

	flight := &models.Flight{
		FlightNumber: flightIATA,
		Airline:      f.Airline.Name,
		Departure:    f.Departure.IATA,
		Arrival:      f.Arrival.IATA,
		Status:       formatStatus(f.FlightStatus),
	}

	if f.Live != nil {
		flight.Latitude = f.Live.Latitude
		flight.Longitude = f.Live.Longitude
		flight.Altitude = f.Live.Altitude * 3.28084   // meters to feet
		flight.Speed = f.Live.SpeedHorizontal * 0.621371 // km/h to mph
	}

	return flight, nil
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
