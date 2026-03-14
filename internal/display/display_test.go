package display

import (
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/xjosh/flightcli/internal/models"
)

func TestAirportBoardTitle(t *testing.T) {
	if got := airportBoardTitle("JFK", "departures"); got != "Departures for JFK" {
		t.Fatalf("unexpected departures title: %q", got)
	}
	if got := airportBoardTitle("JFK", "arrivals"); got != "Arrivals for JFK" {
		t.Fatalf("unexpected arrivals title: %q", got)
	}
}

func TestAirportFlightRow(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	row := airportFlightRow(models.AirportFlight{
		FlightNumber:  "AA100",
		Airline:       "American Airlines",
		Origin:        "JFK",
		Destination:   "LAX",
		Status:        "In Flight",
		ScheduledTime: time.Date(2026, time.March, 14, 15, 30, 0, 0, time.UTC),
	})

	for _, part := range []string{"AA100", "American Airlines", "JFK -> LAX", "In Flight", "15:30"} {
		if !strings.Contains(row, part) {
			t.Fatalf("row %q missing %q", row, part)
		}
	}
}
