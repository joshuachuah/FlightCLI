package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	service       service.FlightService
	screen        screen
	width         int
	height        int
	cursor        int
	focus         int
	inputs        []string
	loading       bool
	activeRequest int
	err           string
	lastUpdated   time.Time
	lastQuery     query
	lastCached    bool
	flight        *models.Flight
	flights       []models.AirportFlight
	activeTitle   string
	statusMessage string
}

var homeItems = []string{
	"Track flight",
	"Airport board",
	"Route search",
	"Quit",
}

const homeBanner = `███████╗██╗     ██╗ ██████╗ ██╗  ██╗████████╗ ██████╗██╗     ██╗
██╔════╝██║     ██║██╔════╝ ██║  ██║╚══██╔══╝██╔════╝██║     ██║
█████╗  ██║     ██║██║  ███╗███████║   ██║   ██║     ██║     ██║
██╔══╝  ██║     ██║██║   ██║██╔══██║   ██║   ██║     ██║     ██║
██║     ███████╗██║╚██████╔╝██║  ██║   ██║   ╚██████╗███████╗██║
╚═╝     ╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝╚══════╝╚═╝`

func Launch(ctx context.Context, svc service.FlightService) error {
	p := tea.NewProgram(initialModel(svc), tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func initialModel(svc service.FlightService) model {
	return model{
		service:       svc,
		screen:        screenHome,
		inputs:        []string{"", "", ""},
		statusMessage: "Use arrow keys to move, Enter to select, q to quit.",
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
		if msg.err != nil {
			m.err = msg.err.Error()
			m.statusMessage = "Request failed. Press esc to go back."
			return m, nil
		}
		m.err = ""
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
				return m, tea.Quit
			case "esc":
				m.loading = false
				m.activeRequest++
				m.statusMessage = "Request dismissed. Press Enter to try again."
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
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
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(homeItems)-1 {
			m.cursor++
		}
	case "enter":
		m.err = ""
		m.inputs = []string{"", "", ""}
		m.focus = 0
		switch m.cursor {
		case 0:
			m.screen = screenFlightForm
			m.statusMessage = "Enter a flight number like AA100, then press Enter."
		case 1:
			m.screen = screenAirportForm
			m.inputs[1] = "departures"
			m.statusMessage = "Enter an airport code, then choose departures or arrivals."
		case 2:
			m.screen = screenSearchForm
			m.statusMessage = "Enter two airport codes to search a route."
		default:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenHome
		m.err = ""
		m.statusMessage = "Use arrow keys to move, Enter to select, q to quit."
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
			return m, fetchQueryCmd(m.service, m.activeRequest, q)
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
			return m, fetchQueryCmd(m.service, m.activeRequest, q)
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
			return m, fetchQueryCmd(m.service, m.activeRequest, q)
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
		m.statusMessage = "Use arrow keys to move, Enter to select, q to quit."
	case "r":
		if m.lastQuery.kind == queryNone {
			return m, nil
		}
		m.activeRequest++
		m.loading = true
		m.err = ""
		m.statusMessage = "Refreshing..."
		return m, fetchQueryCmd(m.service, m.activeRequest, m.lastQuery)
	}
	return m, nil
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

func fetchQueryCmd(svc service.FlightService, requestID int, q query) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

func (m model) View() string {
	switch {
	case m.loading:
		return m.renderFrame("Loading", m.statusMessage+"\n\nPlease wait...")
	case m.screen == screenHome:
		return m.renderFrame("FlightCLI", m.viewHome())
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
	b.WriteString(homeBanner)
	b.WriteString("\n\n")
	b.WriteString("Choose an action:\n\n")
	for i, item := range homeItems {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(cursor)
		b.WriteString(item)
		b.WriteString("\n")
	}
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
	var lines []string
	lines = append(lines, "Flight:   "+flight.FlightNumber)
	lines = append(lines, "Airline:  "+flight.Airline)
	lines = append(lines, "Route:    "+flight.Departure+" -> "+flight.Arrival)
	lines = append(lines, "Status:   "+flight.Status)
	if !flight.DepartureTime.IsZero() {
		lines = append(lines, "Departure: "+flight.DepartureTime.Format(time.RFC1123))
	}
	if !flight.ArrivalTime.IsZero() {
		lines = append(lines, "Arrival:   "+flight.ArrivalTime.Format(time.RFC1123))
	}
	if flight.Latitude != 0 || flight.Longitude != 0 {
		lines = append(lines, fmt.Sprintf("Location: %.4f, %.4f", flight.Latitude, flight.Longitude))
		lines = append(lines, fmt.Sprintf("Altitude: %.0f ft", flight.Altitude))
		lines = append(lines, fmt.Sprintf("Speed:    %.0f mph", flight.Speed))
	}
	return strings.Join(lines, "\n")
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

func trimForWidth(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
