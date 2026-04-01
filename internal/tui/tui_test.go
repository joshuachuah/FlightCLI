package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjosh/flightcli/internal/models"
	"github.com/xjosh/flightcli/internal/provider"
	"github.com/xjosh/flightcli/internal/service"
)

func TestTitleForQuery(t *testing.T) {
	if got := titleForQuery(query{kind: queryFlight, flight: "aa100"}); got != "Flight AA100" {
		t.Fatalf("unexpected flight title: %q", got)
	}
	if got := titleForQuery(query{kind: queryAirport, airport: "jfk", flightType: "arrivals"}); got != "Arrivals for JFK" {
		t.Fatalf("unexpected airport title: %q", got)
	}
	if got := titleForQuery(query{kind: querySearch, from: "jfk", to: "lax"}); got != "Flights from JFK to LAX" {
		t.Fatalf("unexpected search title: %q", got)
	}
}

func TestFormatFlightIncludesRouteAndTelemetry(t *testing.T) {
	output := formatFlight(&models.Flight{
		FlightNumber:  "AA100",
		Airline:       "American Airlines",
		Departure:     "JFK",
		Arrival:       "LAX",
		Status:        "In Flight",
		Altitude:      35000,
		Speed:         510,
		Latitude:      40.7,
		Longitude:     -73.9,
		DepartureTime: time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC),
		ArrivalTime:   time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC),
	})

	for _, part := range []string{"Flight:   AA100", "Route:    JFK -> LAX", "Status:   In Flight", "Altitude: 35000 ft"} {
		if !strings.Contains(output, part) {
			t.Fatalf("flight output %q missing %q", output, part)
		}
	}
}

func TestFormatBoardIncludesRows(t *testing.T) {
	output := formatBoard([]models.AirportFlight{
		{
			FlightNumber:  "DL200",
			Airline:       "Delta Air Lines",
			Origin:        "JFK",
			Destination:   "LAX",
			Status:        "Scheduled",
			ScheduledTime: time.Date(2026, time.April, 1, 16, 45, 0, 0, time.UTC),
		},
	})

	for _, part := range []string{"DL200", "Delta Air Lines", "JFK->LAX", "Scheduled", "16:45"} {
		if !strings.Contains(output, part) {
			t.Fatalf("board output %q missing %q", output, part)
		}
	}
}

func TestTrimForWidth(t *testing.T) {
	if got := trimForWidth("short", 10); got != "short" {
		t.Fatalf("expected untrimmed string, got %q", got)
	}
	if got := trimForWidth("American Airlines", 10); got != "America..." {
		t.Fatalf("expected trimmed string, got %q", got)
	}
}

func TestViewHomeIncludesBanner(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	output := m.viewHome()

	for _, part := range []string{"███████╗██╗", "Choose an action:"} {
		if !strings.Contains(output, part) {
			t.Fatalf("home view %q missing %q", output, part)
		}
	}
}

func TestQIsAcceptedInFlightFormInput(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.screen = screenFlightForm

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatalf("expected form input q not to trigger a command")
	}

	next := updated.(model)
	if next.inputs[0] != "q" {
		t.Fatalf("expected q to be captured in the input field, got %q", next.inputs[0])
	}
}

func TestFetchQueryCmdCancelsContextAfterCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	svc := service.FlightService{
		Provider: &provider.MockProvider{},
	}

	msg := fetchQueryCmd(ctx, cancel, svc, 1, query{kind: queryFlight, flight: "AA100"})()
	if _, ok := msg.(resultPayload); !ok {
		t.Fatalf("expected resultPayload, got %T", msg)
	}

	select {
	case <-ctx.Done():
	default:
		t.Fatalf("expected context to be canceled after command completion")
	}
}

func serviceStub() service.FlightService {
	return service.FlightService{}
}
