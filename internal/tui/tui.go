package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xjosh/flightcli/internal/display"
	"github.com/xjosh/flightcli/internal/models"
	"github.com/xjosh/flightcli/internal/service"
)

type queryKind int

const (
	queryNone queryKind = iota
	queryFlight
	queryAirport
	querySearch
)

type screen int

const (
	screenHome screen = iota
	screenFlightForm
	screenAirportForm
	screenSearchForm
	screenResult
)

type query struct {
	kind       queryKind
	flight     string
	airport    string
	flightType string
	from       string
	to         string
}

type resultPayload struct {
	requestID int
	query     query
	cached    bool
	flight    *models.Flight
	board     []models.AirportFlight
	err       error
}

type model struct {
	appCtx        context.Context
	service       service.FlightService
	screen        screen
	width         int
	height        int
	cursor        int
	focus         int
	commandInput  string
	inputs        []string
	loading       bool
	activeRequest int
	requestCancel context.CancelFunc
	err           string
	lastUpdated   time.Time
	lastQuery     query
	lastCached    bool
	flight        *models.Flight
	flights       []models.AirportFlight
	activeTitle   string
	statusMessage string
}

func Launch(ctx context.Context, svc service.FlightService) error {
	p := tea.NewProgram(initialModel(ctx, svc), tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func initialModel(ctx context.Context, svc service.FlightService) model {
	return model{
		appCtx:        ctx,
		service:       svc,
		screen:        screenHome,
		inputs:        []string{"", "", ""},
		statusMessage: "Type /help for commands. Press q to quit.",
	}
}

func (m model) Init() tea.Cmd {
	return tea.SetWindowTitle("FlightCLI")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case resultPayload:
		if msg.requestID != m.activeRequest {
			return m, nil
		}
		m.loading = false
		m.requestCancel = nil
		if msg.err != nil {
			m.err = msg.err.Error()
			m.statusMessage = "Request failed. Press esc to go back."
			return m, nil
		}
		m.err = ""
		m.commandInput = ""
		m.lastQuery = msg.query
		m.lastCached = msg.cached
		m.flight = msg.flight
		m.flights = msg.board
		m.lastUpdated = time.Now()
		m.activeTitle = titleForQuery(msg.query)
		m.statusMessage = "Press r to refresh, esc to go back, q to quit."
		m.screen = screenResult
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "q":
				if m.requestCancel != nil {
					m.requestCancel()
				}
				return m, tea.Quit
			case "esc":
				m.loading = false
				m.activeRequest++
				if m.requestCancel != nil {
					m.requestCancel()
					m.requestCancel = nil
				}
				m.statusMessage = "Request dismissed. Press Enter to try again."
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenResult || (m.screen == screenHome && m.commandInput == "") {
				return m, tea.Quit
			}
		}

		switch m.screen {
		case screenHome:
			return m.updateHome(msg)
		case screenFlightForm, screenAirportForm, screenSearchForm:
			return m.updateForm(msg)
		case screenResult:
			return m.updateResult(msg)
		}
	}

	return m, nil
}

func (m model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "backspace":
		if m.commandInput != "" {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
	case "enter":
		m.err = ""
		q, done, err := parseSlashCommand(m.commandInput)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if done {
			return m, tea.Quit
		}
		if q.kind == queryNone {
			m.statusMessage = "Type /track [flightNumber], /airport [airport] [departures/arrivals], or /search [airport1] [airport2]. /airport defaults to departures."
			return m, nil
		}
		m.activeRequest++
		m.loading = true
		m.statusMessage = loadingMessage(q)
		return m, m.startRequest(q)
	default:
		if len(msg.Runes) > 0 && !msg.Alt {
			m.commandInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHome
		m.err = ""
		m.statusMessage = "Type /help for commands. Press q to quit."
		return m, nil
	case "tab", "shift+tab", "up", "down":
		m.moveFocus(msg.String())
		return m, nil
	case "backspace":
		if current := m.inputs[m.focus]; current != "" {
			m.inputs[m.focus] = current[:len(current)-1]
		}
		return m, nil
	case "enter":
		switch m.screen {
		case screenFlightForm:
			q := query{kind: queryFlight, flight: strings.TrimSpace(m.inputs[0])}
			if q.flight == "" {
				m.err = "Flight number is required."
				return m, nil
			}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Fetching flight status..."
			return m, m.startRequest(q)
		case screenAirportForm:
			code := strings.TrimSpace(m.inputs[0])
			flightType := strings.TrimSpace(m.inputs[1])
			if code == "" {
				m.err = "Airport code is required."
				return m, nil
			}
			if flightType == "" {
				flightType = "departures"
			}
			q := query{kind: queryAirport, airport: code, flightType: flightType}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Fetching airport board..."
			return m, m.startRequest(q)
		case screenSearchForm:
			q := query{
				kind: querySearch,
				from: strings.TrimSpace(m.inputs[0]),
				to:   strings.TrimSpace(m.inputs[1]),
			}
			if q.from == "" || q.to == "" {
				m.err = "Both airport codes are required."
				return m, nil
			}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Searching route..."
			return m, m.startRequest(q)
		}
	default:
		if len(msg.Runes) > 0 && !msg.Alt {
			m.inputs[m.focus] += string(msg.Runes)
			if m.screen == screenAirportForm && m.focus == 1 {
				m.inputs[m.focus] = strings.ToLower(m.inputs[m.focus])
			}
		}
	}

	return m, nil
}

func (m model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHome
		m.err = ""
		m.statusMessage = "Type /help for commands. Press q to quit."
	case "r":
		if m.lastQuery.kind == queryNone {
			return m, nil
		}
		m.activeRequest++
		m.loading = true
		m.err = ""
		m.statusMessage = "Refreshing..."
		return m, m.startRequest(m.lastQuery)
	}
	return m, nil
}

func (m *model) startRequest(q query) tea.Cmd {
	requestCtx, cancel := context.WithTimeout(m.appCtx, 20*time.Second)
	m.requestCancel = cancel
	return fetchQueryCmd(requestCtx, cancel, m.service, m.activeRequest, q)
}

func (m *model) moveFocus(key string) {
	max := 0
	switch m.screen {
	case screenAirportForm, screenSearchForm:
		max = 1
	default:
		max = 0
	}
	switch key {
	case "up", "shift+tab":
		if m.focus > 0 {
			m.focus--
		} else {
			m.focus = max
		}
	case "down", "tab":
		if m.focus < max {
			m.focus++
		} else {
			m.focus = 0
		}
	}
}

func fetchQueryCmd(ctx context.Context, cancel context.CancelFunc, svc service.FlightService, requestID int, q query) tea.Cmd {
	return func() tea.Msg {
		defer cancel()

		switch q.kind {
		case queryFlight:
			flight, cached, err := svc.GetStatus(ctx, q.flight)
			return resultPayload{requestID: requestID, query: q, flight: flight, cached: cached, err: err}
		case queryAirport:
			flights, cached, err := svc.GetAirportFlights(ctx, q.airport, q.flightType)
			return resultPayload{requestID: requestID, query: q, board: flights, cached: cached, err: err}
		case querySearch:
			flights, cached, err := svc.SearchFlights(ctx, q.from, q.to)
			return resultPayload{requestID: requestID, query: q, board: flights, cached: cached, err: err}
		default:
			return resultPayload{requestID: requestID, query: q, err: fmt.Errorf("unsupported query")}
		}
	}
}

func parseSlashCommand(input string) (query, bool, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return query{}, false, nil
	}
	if !strings.HasPrefix(trimmed, "/") {
		return query{}, false, fmt.Errorf("commands must start with /")
	}

	fields := strings.Fields(trimmed)
	command := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	args := fields[1:]

	switch command {
	case "track", "flight", "status":
		if len(args) != 1 {
			return query{}, false, fmt.Errorf("usage: /track AA100")
		}
		return query{kind: queryFlight, flight: args[0]}, false, nil
	case "airport", "board":
		if len(args) < 1 || len(args) > 2 {
			return query{}, false, fmt.Errorf("usage: /airport JFK departures")
		}
		flightType := "departures"
		if len(args) == 2 {
			flightType = strings.ToLower(args[1])
		}
		if flightType != "departures" && flightType != "arrivals" {
			return query{}, false, fmt.Errorf("airport board type must be departures or arrivals")
		}
		return query{kind: queryAirport, airport: args[0], flightType: flightType}, false, nil
	case "search", "route":
		if len(args) != 2 {
			return query{}, false, fmt.Errorf("usage: /search JFK LAX")
		}
		return query{kind: querySearch, from: args[0], to: args[1]}, false, nil
	case "help":
		return query{}, false, nil
	case "quit", "exit":
		return query{}, true, nil
	default:
		return query{}, false, fmt.Errorf("unknown command %q", fields[0])
	}
}

func loadingMessage(q query) string {
	switch q.kind {
	case queryFlight:
		return "Fetching flight status..."
	case queryAirport:
		return "Fetching airport board..."
	case querySearch:
		return "Searching route..."
	default:
		return "Loading..."
	}
}

func (m model) View() string {
	switch {
	case m.loading:
		return m.renderFrame("Loading", "Please wait...")
	case m.screen == screenHome:
		return m.renderFrame("", m.viewHome())
	case m.screen == screenFlightForm:
		return m.renderFrame("Track Flight", m.viewFlightForm())
	case m.screen == screenAirportForm:
		return m.renderFrame("Airport Board", m.viewAirportForm())
	case m.screen == screenSearchForm:
		return m.renderFrame("Route Search", m.viewSearchForm())
	case m.screen == screenResult:
		return m.renderFrame(m.activeTitle, m.viewResult())
	default:
		return m.renderFrame("FlightCLI", "")
	}
}

func (m model) renderFrame(title, body string) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(title)
		b.WriteString("\n")
		b.WriteString(strings.Repeat("=", max(12, len(title))))
		b.WriteString("\n\n")
	}
	if m.err != "" {
		b.WriteString("Error: ")
		b.WriteString(m.err)
		b.WriteString("\n\n")
	}
	b.WriteString(body)
	if m.statusMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(m.statusMessage)
	}
	return b.String()
}

func (m model) viewHome() string {
	var b strings.Builder
	b.WriteString("Command:\n")
	b.WriteString("> ")
	b.WriteString(m.commandInput)
	b.WriteString("_\n\n")
	b.WriteString("Commands:\n")
	b.WriteString("  /track [flightNumber]\n")
	b.WriteString("  /airport [airport] [departures]\n")
	b.WriteString("  /airport [airport] [arrivals]\n")
	b.WriteString("  /search [airport1] [airport2]\n")
	b.WriteString("  /quit\n")
	return b.String()
}

func (m model) viewFlightForm() string {
	return m.renderInputs([]field{
		{label: "Flight number", value: strings.ToUpper(m.inputs[0]), hint: "Example: AA100"},
	})
}

func (m model) viewAirportForm() string {
	flightType := m.inputs[1]
	if flightType == "" {
		flightType = "departures"
	}
	return m.renderInputs([]field{
		{label: "Airport code", value: strings.ToUpper(m.inputs[0]), hint: "Example: JFK"},
		{label: "Board type", value: strings.ToLower(flightType), hint: "departures or arrivals"},
	})
}

func (m model) viewSearchForm() string {
	return m.renderInputs([]field{
		{label: "From", value: strings.ToUpper(m.inputs[0]), hint: "Example: JFK"},
		{label: "To", value: strings.ToUpper(m.inputs[1]), hint: "Example: LAX"},
	})
}

func (m model) viewResult() string {
	var b strings.Builder
	if m.lastCached {
		b.WriteString("(cached)\n\n")
	}
	if m.flight != nil {
		b.WriteString(formatFlight(m.flight))
	} else if m.lastQuery.kind == querySearch {
		b.WriteString(formatSearchResults(m.flights))
	} else {
		b.WriteString(formatBoard(m.flights))
	}
	if !m.lastUpdated.IsZero() {
		b.WriteString("\n\nLast updated: ")
		b.WriteString(m.lastUpdated.Format("15:04:05"))
	}
	return b.String()
}

type field struct {
	label string
	value string
	hint  string
}

func (m model) renderInputs(fields []field) string {
	var b strings.Builder
	for i, f := range fields {
		cursor := "  "
		if i == m.focus {
			cursor = "> "
		}
		b.WriteString(cursor)
		b.WriteString(f.label)
		b.WriteString(": ")
		b.WriteString(f.value)
		if i == m.focus {
			b.WriteString("_")
		}
		b.WriteString("\n")
		b.WriteString("    ")
		b.WriteString(f.hint)
		b.WriteString("\n\n")
	}
	b.WriteString("Tab moves between fields. Enter submits. Esc goes back.")
	return b.String()
}

func titleForQuery(q query) string {
	switch q.kind {
	case queryFlight:
		return "Flight " + strings.ToUpper(q.flight)
	case queryAirport:
		label := capitalize(strings.ToLower(q.flightType))
		return label + " for " + strings.ToUpper(q.airport)
	case querySearch:
		return "Flights from " + strings.ToUpper(q.from) + " to " + strings.ToUpper(q.to)
	default:
		return "FlightCLI"
	}
}

func formatFlight(flight *models.Flight) string {
	return formatFlightAt(flight, time.Now())
}

func formatBoard(flights []models.AirportFlight) string {
	if len(flights) == 0 {
		return "No flights found."
	}

	var lines []string
	for _, f := range flights {
		scheduled := ""
		if !f.ScheduledTime.IsZero() {
			scheduled = f.ScheduledTime.Format("15:04")
		}
		lines = append(lines, fmt.Sprintf("%-8s %-22s %-11s %-10s %s",
			f.FlightNumber,
			trimForWidth(f.Airline, 22),
			f.Origin+"->"+f.Destination,
			trimForWidth(f.Status, 10),
			scheduled,
		))
	}
	return strings.Join(lines, "\n")
}

func formatFlightAt(flight *models.Flight, now time.Time) string {
	return strings.Join(display.FlightStatusLines(flight, now), "\n")
}

func formatSearchResults(flights []models.AirportFlight) string {
	if len(flights) == 0 {
		return "No flights found."
	}

	sections := make([]string, 0, len(flights))
	for _, flight := range flights {
		sections = append(sections, strings.Join(display.SearchFlightLines(flight), "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func trimForWidth(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	runes := []rune(s)
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
