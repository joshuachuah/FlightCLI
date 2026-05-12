package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joshuachuah/flightcli/internal/models"
	"github.com/joshuachuah/flightcli/internal/provider"
	"github.com/joshuachuah/flightcli/internal/service"
)

func TestTitleForQuery(t *testing.T) {
	if got := titleForQuery(query{kind: queryFlight, flight: "aa100"}); got != "Flight AA100" {
		t.Fatalf("unexpected flight title: %q", got)
	}
	if got := titleForQuery(query{kind: queryAirport, airport: "jfk", flightType: "arrivals"}); got != "Arrivals for JFK" {
		t.Fatalf("unexpected airport title: %q", got)
	}
	if got := titleForQuery(query{kind: querySearch, from: "jfk", to: "lax"}); got != "JFK → LAX" {
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

func TestFormatBoardSanitizesTerminalControls(t *testing.T) {
	output := formatBoard([]models.AirportFlight{
		{
			FlightNumber: "DL200\x1b[2J",
			Airline:      "Delta\u009d0;spoof\a Air Lines",
			Origin:       "JFK",
			Destination:  "LAX\x00",
			Status:       "Scheduled\x1b]8;;https://evil.test\a\x1b]8;;\a",
		},
	})

	for _, forbidden := range []string{"\x1b[2J", "\a", "\x00", "spoof", "https://evil.test"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("board output %q still contains forbidden terminal content %q", output, forbidden)
		}
	}
	for _, part := range []string{"DL200", "Delta Air Lines", "JFK->LAX", "Scheduled"} {
		if !strings.Contains(output, part) {
			t.Fatalf("board output %q missing sanitized content %q", output, part)
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
	if got := trimForWidth("München", 5); got != "Mü..." {
		t.Fatalf("expected trimmed at rune boundary, got %q", got)
	}
	// CJK: each character is 3 bytes but 1 rune
	if got := trimForWidth("东京成田", 4); got != "东京成田" {
		t.Fatalf("expected full CJK string, got %q", got)
	}
	if got := trimForWidth("东京成田", 2); got != "东京" {
		t.Fatalf("expected CJK trimmed at rune boundary, got %q", got)
	}
}

func TestViewHomeShowsErrorInScrollback(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.width = 80
	m.height = 24
	m.err = "something went wrong"
	m.scrollback = []string{m.renderErrorBlock()}

	output := m.View()
	if !strings.Contains(output, "something went wrong") {
		t.Fatalf("expected view to show error message in scrollback, got:\n%s", output)
	}

	// Home view without error or scrollback should be mostly empty
	m.err = ""
	m.scrollback = nil
	output = m.View()
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected status bar and input bar lines, got:\n%s", output)
	}
	if !strings.Contains(lines[m.height-2], ">") {
		t.Fatalf("expected input line pinned above hint at bottom, got line %q in:\n%s", lines[m.height-2], output)
	}
	if !strings.Contains(lines[m.height-1], "? for shortcuts") {
		t.Fatalf("expected input hint pinned to bottom, got line %q in:\n%s", lines[m.height-1], output)
	}
	for _, cmd := range []string{"/track", "/airport", "/search"} {
		if strings.Contains(output, cmd) {
			t.Fatalf("expected minimal home view without command menu, but found %q", cmd)
		}
	}
}

func TestViewOverflowShowsLatestContentWithInputAtBottom(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.width = 80
	m.height = 10
	m.scrollback = []string{strings.Join([]string{
		"line 01",
		"line 02",
		"line 03",
		"line 04",
		"line 05",
		"line 06",
		"line 07",
		"line 08",
		"line 09",
		"line 10",
	}, "\n")}

	output := m.View()
	lines := strings.Split(output, "\n")
	if len(lines) != m.height {
		t.Fatalf("expected view to render %d lines, got %d:\n%s", m.height, len(lines), output)
	}
	if strings.Contains(output, "line 01") {
		t.Fatalf("expected overflowing content to start scrolled toward latest lines, got:\n%s", output)
	}
	if !strings.Contains(output, "line 10") {
		t.Fatalf("expected overflowing content to include latest line, got:\n%s", output)
	}
	if !strings.Contains(lines[m.height-2], ">") {
		t.Fatalf("expected input line pinned above hint at bottom, got line %q in:\n%s", lines[m.height-2], output)
	}
	if !strings.Contains(lines[m.height-1], "? for shortcuts") {
		t.Fatalf("expected input hint pinned to bottom, got line %q in:\n%s", lines[m.height-1], output)
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
	if next.screen != screenHome {
		t.Fatalf("expected successful result to return to home screen, got %v", next.screen)
	}
	if len(next.scrollback) == 0 {
		t.Fatalf("expected result to be appended to scrollback")
	}
}

func TestLoadingViewShowsStatus(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.width = 80
	m.height = 24
	m.loading = true
	m.statusMessage = "Fetching flight status..."

	output := m.View()
	if !strings.Contains(output, "Fetching flight status...") {
		t.Fatalf("expected loading view to include status message, got %q", output)
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

func TestSpinnerAlwaysSchedulesNextTick(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.loading = false // Not loading

	updated, cmd := m.Update(spinnerTickMsg{})
	if cmd == nil {
		t.Fatalf("expected spinnerTick to always schedule next tick, even when not loading")
	}
	// Spinner frame should NOT advance when not loading
	next := updated.(model)
	if next.spinnerFrame != 0 {
		t.Fatalf("expected spinner frame to stay 0 when not loading, got %d", next.spinnerFrame)
	}

	// Now test that it DOES advance when loading
	m2 := initialModel(context.Background(), serviceStub())
	m2.loading = true
	updated2, cmd2 := m2.Update(spinnerTickMsg{})
	if cmd2 == nil {
		t.Fatalf("expected spinnerTick to schedule next tick when loading")
	}
	next2 := updated2.(model)
	if next2.spinnerFrame != 1 {
		t.Fatalf("expected spinner frame to advance when loading, got %d", next2.spinnerFrame)
	}
}

func TestErrorAutoDismissScopedByID(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())

	// Set first error
	cmd1 := m.setError("first error")
	if m.err != "first error" {
		t.Fatalf("expected first error message, got %q", m.err)
	}
	firstID := m.errID

	// Set second error (should get a new errID)
	cmd2 := m.setError("second error")
	if m.err != "second error" {
		t.Fatalf("expected second error message, got %q", m.err)
	}
	secondID := m.errID
	if secondID <= firstID {
		t.Fatalf("expected errID to increment, got first=%d second=%d", firstID, secondID)
	}

	// Simulate the first error's clear timer firing — should NOT clear the second error
	updated, _ := m.Update(clearErrorMsg{id: firstID})
	next := updated.(model)
	if next.err != "second error" {
		t.Fatalf("expected old clearErrorMsg not to wipe newer error, got %q", next.err)
	}
	_ = cmd1
	_ = cmd2

	// Simulate the second error's clear timer — SHOULD clear the error
	updated2, _ := next.Update(clearErrorMsg{id: secondID})
	next2 := updated2.(model)
	if next2.err != "" {
		t.Fatalf("expected matching clearErrorMsg to clear error, got %q", next2.err)
	}
}

func TestHelpScreenView(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.width = 80
	m.height = 24
	m.screen = screenHelp
	m.statusMessage = "any key back"

	output := m.View()
	for _, part := range []string{"/track", "/airport", "/search", "/help", "Keyboard Shortcuts"} {
		if !strings.Contains(output, part) {
			t.Fatalf("expected help view to contain %q, got:\n%s", part, output)
		}
	}
}

func TestTabCompletionCyclesMatches(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/tr"

	// First tab — should complete to "/track "
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	next := updated.(model)
	if next.commandInput != "/track " {
		t.Fatalf("expected tab completion to yield '/track ', got %q", next.commandInput)
	}

	// Second tab — should cycle to next match (e.g. nothing else starts with /tr after removing /track)
	// Actually /track is the only /tr prefix command, so it stays
	updated2, _ := next.Update(tea.KeyMsg{Type: tea.KeyTab})
	next2 := updated2.(model)
	// It should cycle back to "/track " since it's the only match
	if next2.commandInput != "/track " {
		t.Fatalf("expected tab cycling to stay on '/track ', got %q", next2.commandInput)
	}
}

func TestSearchHistoryNavigation(t *testing.T) {
	m := initialModel(context.Background(), serviceStub())
	m.commandInput = "/"
	m.history = []string{"/track AA100", "/airport JFK", "/search JFK LAX"}

	// Up arrow should go to most recent history
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	next := updated.(model)
	if next.commandInput != "/search JFK LAX" {
		t.Fatalf("expected up arrow to recall most recent history, got %q", next.commandInput)
	}

	// Up again — earlier entry
	updated2, _ := next.Update(tea.KeyMsg{Type: tea.KeyUp})
	next2 := updated2.(model)
	if next2.commandInput != "/airport JFK" {
		t.Fatalf("expected second up arrow to go to earlier history, got %q", next2.commandInput)
	}

	// Down arrow — go back to more recent
	updated3, _ := next2.Update(tea.KeyMsg{Type: tea.KeyDown})
	next3 := updated3.(model)
	if next3.commandInput != "/search JFK LAX" {
		t.Fatalf("expected down arrow to go to more recent history, got %q", next3.commandInput)
	}

	// Down arrow at end — clear input
	updated4, _ := next3.Update(tea.KeyMsg{Type: tea.KeyDown})
	next4 := updated4.(model)
	if next4.commandInput != "" {
		t.Fatalf("expected down arrow at end of history to clear input, got %q", next4.commandInput)
	}
}
