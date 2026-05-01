package tui

import (
	"context"
	"fmt"
	"regexp"
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
	spinnerFrame  int
}

var airportCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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
		statusMessage: "Type /help for commands",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("FlightCLI"),
		spinnerTick(),
	)
}

type spinnerTickMsg struct{}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinnerTickMsg:
		if m.loading {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, spinnerTick()
		}
		return m, nil
	case resultPayload:
		if msg.requestID != m.activeRequest {
			return m, nil
		}
		m.loading = false
		m.requestCancel = nil
		if msg.err != nil {
			m.err = msg.err.Error()
			m.statusMessage = "Request failed"
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
		m.statusMessage = "r refresh · esc back · q quit"
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
				m.statusMessage = "Dismissed"
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
			m.statusMessage = "/track [flight] · /airport [code] · /search [from] [to]"
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
		m.statusMessage = "Type /help for commands"
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
				m.err = "Flight number is required"
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
				m.err = "Airport code is required"
				return m, nil
			}
			if !airportCodePattern.MatchString(strings.ToUpper(code)) {
				m.err = "Invalid airport code: use 3-letter IATA"
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
				m.err = "Both airport codes are required"
				return m, nil
			}
			if !airportCodePattern.MatchString(strings.ToUpper(q.from)) {
				m.err = "Invalid origin: use 3-letter IATA"
				return m, nil
			}
			if !airportCodePattern.MatchString(strings.ToUpper(q.to)) {
				m.err = "Invalid destination: use 3-letter IATA"
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
		m.statusMessage = "Type /help for commands"
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