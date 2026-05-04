package display

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/joshuachuah/flightcli/internal/models"
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
				Status:        "In Flight",
				Latitude:      40.7,
				Longitude:     -73.9,
				Altitude:      35000,
				Speed:         510,
				DepartureTime: time.Date(2026, time.March, 14, 16, 45, 0, 0, time.UTC),
				ArrivalTime:   time.Date(2026, time.March, 14, 20, 30, 0, 0, time.UTC),
			},
		}, "JFK", "LAX")
	})

	for _, part := range []string{"Flights from JFK to LAX", "Departure:", "Arrival:", "Location:", "Altitude:", "Speed:"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search output %q missing %q", output, part)
		}
	}
}

func TestFlightStatusLinesIncludesElapsedAndRemaining(t *testing.T) {
	departure := time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC)
	arrival := time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC)

	lines := FlightStatusLines(&models.Flight{
		FlightNumber:  "AA100",
		Airline:       "American Airlines",
		Departure:     "JFK",
		Arrival:       "LAX",
		Status:        "In Flight",
		DepartureTime: departure,
		ArrivalTime:   arrival,
	}, departure.Add(90*time.Minute))

	output := strings.Join(lines, "\n")
	for _, part := range []string{"Flight Time: 3h 0m", "Time Elapsed: 1h 30m", "Time to Destination: 1h 30m"} {
		if !strings.Contains(output, part) {
			t.Fatalf("flight lines %q missing %q", output, part)
		}
	}
}

func TestFlightStatusLinesSanitizesTerminalControls(t *testing.T) {
	lines := FlightStatusLines(&models.Flight{
		FlightNumber: "AA100\x1b[31m",
		Airline:      "Safe Air\x1b]8;;https://evil.test\aLink\x1b]8;;\a",
		Departure:    "JFK\r",
		Arrival:      "LAX",
		Status:       "In Flight\x1b]0;spoof\x1b\\",
	}, time.Now())

	output := strings.Join(lines, "\n")
	for _, forbidden := range []string{"\x1b", "\r", "\a", "https://evil.test", "spoof"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("flight lines %q still contain forbidden terminal content %q", output, forbidden)
		}
	}
	for _, part := range []string{"AA100", "Safe AirLink", "Route:    JFK -> LAX", "Status:   In Flight"} {
		if !strings.Contains(output, part) {
			t.Fatalf("flight lines %q missing sanitized content %q", output, part)
		}
	}
}

func TestPrintFlightStatusSanitizesTerminalControls(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	output := captureStdout(t, func() {
		PrintFlightStatus(&models.Flight{
			FlightNumber: "AA100\x1b[31m",
			Airline:      "Safe Air\x1b]0;spoof\a Lines",
			Departure:    "JFK\u009b2J",
			Arrival:      "LAX\r",
			Status:       "In Flight\x1b]8;;https://evil.test\aLink\x1b]8;;\a",
		})
	})

	for _, forbidden := range []string{"\x1b", "\a", "\r", "spoof", "https://evil.test"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("flight output %q still contains forbidden terminal content %q", output, forbidden)
		}
	}
	for _, part := range []string{"Flight:   AA100", "Airline:  Safe Air Lines", "Route:    JFK -> LAX", "Status:   In FlightLink"} {
		if !strings.Contains(output, part) {
			t.Fatalf("flight output %q missing sanitized content %q", output, part)
		}
	}
}

func TestSearchFlightLinesIncludeRestoredMetrics(t *testing.T) {
	lines := SearchFlightLines(models.AirportFlight{
		FlightNumber:  "DL200",
		Airline:       "Delta Airlines",
		Origin:        "JFK",
		Destination:   "LAX",
		Status:        "In Flight",
		Latitude:      40.7128,
		Longitude:     -73.9352,
		Altitude:      34500,
		Speed:         515,
		DepartureTime: time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC),
		ArrivalTime:   time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC),
	})

	output := strings.Join(lines, "\n")
	for _, part := range []string{"Departure:", "Arrival:", "Location:", "Altitude:", "Speed:"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search lines %q missing %q", output, part)
		}
	}
}

func TestSearchFlightLinesIncludeZeroTelemetryWhenLocationExists(t *testing.T) {
	lines := SearchFlightLines(models.AirportFlight{
		FlightNumber: "DL201",
		Airline:      "Delta Airlines",
		Origin:       "JFK",
		Destination:  "LAX",
		Status:       "In Flight",
		Latitude:     40.7128,
		Longitude:    -73.9352,
		Altitude:     0,
		Speed:        0,
	})

	output := strings.Join(lines, "\n")
	for _, part := range []string{"Location:", "Altitude:  0 ft", "Speed:     0 mph"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search lines %q missing %q", output, part)
		}
	}
}

func TestAirportFlightRowSanitizesTerminalControls(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	row := airportFlightRow(models.AirportFlight{
		FlightNumber: "DL200\x1b[2J",
		Airline:      "Delta\x1b]0;spoof\x1b\\ Air Lines",
		Origin:       "JFK",
		Destination:  "LAX\x00",
		Status:       "Scheduled\x1b[31m",
	})

	for _, forbidden := range []string{"\x1b", "\x00", "spoof"} {
		if strings.Contains(row, forbidden) {
			t.Fatalf("airport row %q still contains forbidden terminal content %q", row, forbidden)
		}
	}
	for _, part := range []string{"DL200", "Delta Air Lines", "JFK -> LAX", "Scheduled"} {
		if !strings.Contains(row, part) {
			t.Fatalf("airport row %q missing sanitized content %q", row, part)
		}
	}
}

func TestPrintSearchResultsIncludeZeroTelemetryWhenLocationExists(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	output := captureStdout(t, func() {
		PrintSearchResults([]models.AirportFlight{
			{
				FlightNumber: "DL201",
				Airline:      "Delta Airlines",
				Origin:       "JFK",
				Destination:  "LAX",
				Status:       "In Flight",
				Latitude:     40.7128,
				Longitude:    -73.9352,
				Altitude:     0,
				Speed:        0,
			},
		}, "JFK", "LAX")
	})

	for _, part := range []string{"Location:", "Altitude:", "0 ft", "Speed:", "0 mph"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search output %q missing %q", output, part)
		}
	}
}
