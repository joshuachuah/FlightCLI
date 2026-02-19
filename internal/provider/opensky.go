package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/xjosh/flightcli/internal/models"
)

type OpenSkyProvider struct{}

type openSkyResponse struct {
	Time   int             `json:"time"`
	States [][]interface{} `json:"states"`
}

func (o *OpenSkyProvider) GetFlightStatus(flightNumber string) (*models.Flight, error) {
	callsign := strings.ToUpper(strings.TrimSpace(flightNumber))

	url := fmt.Sprintf("https://opensky-network.org/api/states/all?callsign=%s", callsign)

	resp, err := providerHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach OpenSky API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenSky API returned status %d", resp.StatusCode)
	}

	var data openSkyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(data.States) == 0 {
		return nil, fmt.Errorf("no flight found for callsign %s", callsign)
	}

	state := data.States[0]

	flight := &models.Flight{
		FlightNumber: callsign,
		Airline:      stringVal(state, 2),
		Status:       deriveStatus(state),
		Latitude:     floatVal(state, 6),
		Longitude:    floatVal(state, 5),
		Altitude:     metersToFeet(floatVal(state, 7)),
		Speed:        msToMph(floatVal(state, 9)),
	}

	return flight, nil
}

func (o *OpenSkyProvider) GetAirportFlights(airportCode string, flightType string) ([]models.AirportFlight, error) {
	return nil, fmt.Errorf("GetAirportFlights is not supported by the OpenSky provider")
}

func (o *OpenSkyProvider) SearchFlights(from, to string) ([]models.AirportFlight, error) {
	return nil, fmt.Errorf("SearchFlights is not supported by the OpenSky provider")
}

func deriveStatus(state []interface{}) string {
	if len(state) > 8 {
		if onGround, ok := state[8].(bool); ok && onGround {
			return "On Ground"
		}
	}
	return "In Flight"
}

func stringVal(state []interface{}, idx int) string {
	if idx < len(state) {
		if s, ok := state[idx].(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return "Unknown"
}

func floatVal(state []interface{}, idx int) float64 {
	if idx < len(state) {
		if f, ok := state[idx].(float64); ok {
			return f
		}
	}
	return 0
}

func metersToFeet(m float64) float64 {
	return m * 3.28084
}

func msToMph(ms float64) float64 {
	return ms * 2.23694
}
