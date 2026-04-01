package display

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/xjosh/flightcli/internal/models"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	originalColorOutput := color.Output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = w
	color.Output = w
	t.Cleanup(func() {
		os.Stdout = originalStdout
		color.Output = originalColorOutput
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}

	return string(out)
}

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

func TestPrintAirportFlightsUsesSharedRowFormatting(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	output := captureStdout(t, func() {
		PrintAirportFlights([]models.AirportFlight{
			{
				FlightNumber:  "AA100",
				Airline:       "American Airlines",
				Origin:        "JFK",
				Destination:   "LAX",
				Status:        "In Flight",
				ScheduledTime: time.Date(2026, time.March, 14, 15, 30, 0, 0, time.UTC),
			},
		}, "JFK", "departures")
	})

	for _, part := range []string{"Departures for JFK", "JFK -> LAX", "In Flight", "15:30"} {
		if !strings.Contains(output, part) {
			t.Fatalf("airport output %q missing %q", output, part)
		}
	}
}

func TestPrintSearchResultsUsesSharedRowFormatting(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	output := captureStdout(t, func() {
		PrintSearchResults([]models.AirportFlight{
			{
				FlightNumber:  "DL200",
				Airline:       "Delta Airlines",
				Origin:        "JFK",
				Destination:   "LAX",
				Status:        "Scheduled",
				ScheduledTime: time.Date(2026, time.March, 14, 16, 45, 0, 0, time.UTC),
			},
		}, "JFK", "LAX")
	})

	for _, part := range []string{"Flights from JFK to LAX", "JFK -> LAX", "Scheduled", "16:45"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search output %q missing %q", output, part)
		}
	}
}
