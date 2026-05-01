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
	departure := time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC)
	arrival := time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC)

	output := formatFlightAt(&models.Flight{
		FlightNumber:  "AA100",
		Airline:       "American Airlines",
		Departure:     "JFK",
		Arrival:       "LAX",
		Status:        "In Flight",
		Altitude:      35000,
		Speed:         510,
		Latitude:      40.7,
		Longitude:     -73.9,
		DepartureTime: departure,
		ArrivalTime:   arrival,
	}, departure.Add(90*time.Minute))

	for _, part := range []string{"Flight:   AA100", "Route:    JFK -> LAX", "Status:   In Flight", "Altitude: 35000 ft", "Time Elapsed: 1h 30m", "Time to Destination: 1h 30m"} {
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

func TestFormatSearchResultsIncludesRestoredMetrics(t *testing.T) {
	output := formatSearchResults([]models.AirportFlight{
		{
			FlightNumber:  "DL200",
			Airline:       "Delta Air Lines",
			Origin:        "JFK",
			Destination:   "LAX",
			Status:        "In Flight",
			Latitude:      40.7,
			Longitude:     -73.9,
			Altitude:      35000,
			Speed:         510,
			DepartureTime: time.Date(2026, time.April, 1, 8, 0, 0, 0, time.UTC),
			ArrivalTime:   time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC),
		},
	})

	for _, part := range []string{"Departure:", "Arrival:", "Location:", "Altitude:", "Speed:"} {
		if !strings.Contains(output, part) {
			t.Fatalf("search output %q missing %q", output, part)
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

func TestTrimForWidthHandlesMultibyteRunes(t *testing.T) {
	// "München" is 7 runes but 8 bytes (ü is 2 bytes)
	if got := trimForWidth("München", 7); got != "München" {
		t.Fatalf("expected full string, got %q", got)
	}
	if got := trimForWidth("München", 5); got != "Münc..." {
		t.Fatalf("expected trimmed at rune boundary, got %q", got)
	}
	// CJK: each character is 3 bytes but 1 rune
	if got := trimForWidth("东京成田", 4); got != "东京成田" {
		t.Fatalf("expected full CJK string, got %q", got)
	}
	if got := trimForWidth("东京成田", 2); got != "东京..." {
		t.Fatalf("expected CJK trimmed at rune boundary, got %q", got)
	}
}

func TestViewHomeShowsSlashCommandsWithoutTitle(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	output := m.View()

	for _, part := range []string{"Command:", "/track [flightNumber]", "/airport [airport] [departures]", "/search [airport1] [airport2]"} {
		if !strings.Contains(output, part) {
			t.Fatalf("home view %q missing %q", output, part)
		}
	}
	for _, part := range []string{"FlightCLI\n============", "Choose an action:"} {
		if strings.Contains(output, part) {
			t.Fatalf("home view %q should not contain %q", output, part)
		}
	}
}

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  query
	}{
		{
			name:  "track",
			input: "/track AA100",
			want:  query{kind: queryFlight, flight: "AA100"},
		},
		{
			name:  "airport defaults to departures",
			input: "/airport JFK",
			want:  query{kind: queryAirport, airport: "JFK", flightType: "departures"},
		},
		{
			name:  "airport arrivals",
			input: "/airport JFK arrivals",
			want:  query{kind: queryAirport, airport: "JFK", flightType: "arrivals"},
		},
		{
			name:  "search",
			input: "/search JFK LAX",
			want:  query{kind: querySearch, from: "JFK", to: "LAX"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, done, err := parseSlashCommand(tt.input)
			if err != nil {
				t.Fatalf("parseSlashCommand returned error: %v", err)
			}
			if done {
				t.Fatalf("parseSlashCommand returned done=true")
			}
			if got != tt.want {
				t.Fatalf("parseSlashCommand returned %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestHomeSlashCommandStartsRequest(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/search JFK LAX"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command to start request")
	}

	next := updated.(model)
	if !next.loading {
		t.Fatalf("expected slash command to put model in loading state")
	}
	if next.commandInput != "/search JFK LAX" {
		t.Fatalf("expected command input to stay available while loading, got %q", next.commandInput)
	}
	if next.statusMessage != "Searching route..." {
		t.Fatalf("unexpected status message %q", next.statusMessage)
	}
}

func TestHomeSlashCommandCanRetryAfterLoadingCancel(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/track AA100"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command to start request")
	}

	loading := updated.(model)
	canceled, cmd := loading.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected canceling loading request not to return a command")
	}

	afterCancel := canceled.(model)
	if afterCancel.loading {
		t.Fatalf("expected loading to be false after cancel")
	}
	if afterCancel.commandInput != "/track AA100" {
		t.Fatalf("expected command input to be preserved after cancel, got %q", afterCancel.commandInput)
	}

	retried, cmd := afterCancel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected preserved command to retry request")
	}
	if !retried.(model).loading {
		t.Fatalf("expected retry to put model back in loading state")
	}
}

func TestHomeSlashCommandClearsAfterSuccessfulResult(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/track AA100"
	m.loading = true
	m.activeRequest = 1

	updated, cmd := m.Update(resultPayload{
		requestID: 1,
		query:     query{kind: queryFlight, flight: "AA100"},
		flight:    &models.Flight{FlightNumber: "AA100"},
	})
	if cmd != nil {
		t.Fatalf("expected successful result not to return a command")
	}

	next := updated.(model)
	if next.commandInput != "" {
		t.Fatalf("expected command input to clear after successful result, got %q", next.commandInput)
	}
	if next.screen != screenResult {
		t.Fatalf("expected successful result to show result screen, got %v", next.screen)
	}
}

func TestLoadingViewShowsStatusOnce(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.loading = true
	m.statusMessage = "Fetching flight status..."

	output := m.View()
	if count := strings.Count(output, "Fetching flight status..."); count != 1 {
		t.Fatalf("expected loading status once, got %d in %q", count, output)
	}
	if !strings.Contains(output, "Please wait...") {
		t.Fatalf("expected loading view to include wait message, got %q", output)
	}
}

func TestQCanBeTypedInsideSlashCommand(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatalf("expected q in command input not to quit")
	}

	next := updated.(model)
	if next.commandInput != "/q" {
		t.Fatalf("expected q to be captured in command input, got %q", next.commandInput)
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
