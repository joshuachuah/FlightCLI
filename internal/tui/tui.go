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
	queryHelp
)

type screen int

const (
	screenHome screen = iota
	screenFlightForm
	screenAirportForm
	screenSearchForm
	screenResult
	screenHelp
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

type clearErrorMsg struct {
	id int
}

type model struct {
	appCtx            context.Context
	service           service.FlightService
	screen            screen
	width             int
	height            int
	cursor            int
	focus             int
	commandInput      string
	inputs            []string
	loading           bool
	activeRequest     int
	requestCancel     context.CancelFunc
	err               string
	errID             int
	scrollback        []string
	scrollOffset      int
	lastUpdated       time.Time
	lastQuery         query
	lastCached        bool
	flight            *models.Flight
	flights           []models.AirportFlight
	activeTitle       string
	statusMessage     string
	spinnerFrame      int
	history           []string
	historyIndex      int
	completionBase    string
	completionMatches []string
	completionIndex   int
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
		appCtx:          ctx,
		service:         svc,
		screen:          screenHome,
		inputs:          []string{"", "", ""},
		statusMessage:   "Type /help for commands",
		historyIndex:    -1,
		completionIndex: -1,
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
		}
		return m, spinnerTick()
	case resultPayload:
		if msg.requestID != m.activeRequest {
			return m, nil
		}
		m.loading = false
		m.requestCancel = nil
		if msg.err != nil {
			cmd := m.setError(msg.err.Error())
			m.statusMessage = "Request failed"
			// Append error to scrollback
			m.scrollback = append(m.scrollback, m.renderErrorBlock())
			m.screen = screenHome
			m.scrollOffset = 0
			m.clampScroll()
			return m, cmd
		}
		m.err = ""
		m.commandInput = ""
		m.lastQuery = msg.query
		m.lastCached = msg.cached
		m.flight = msg.flight
		m.flights = msg.board
		m.lastUpdated = time.Now()
		m.activeTitle = titleForQuery(msg.query)
		m.statusMessage = "Type /help for commands"
		// Append result to scrollback
		m.scrollback = append(m.scrollback, m.renderResultBlock())
		m.screen = screenHome
		m.scrollOffset = 0
		m.clampScroll()
		return m, nil
	case clearErrorMsg:
		if msg.id == m.errID {
			m.err = ""
		}
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

		// Scroll handling (works from home and help screens)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.screen == screenHome && m.commandInput == "" {
				return m, tea.Quit
			}
		case "up":
			if m.screen == screenHome && m.commandInput == "" && m.maxScrollOffset() > 0 {
				m.scrollOffset++
				m.clampScroll()
				return m, nil
			}
		case "down":
			if m.screen == screenHome && m.commandInput == "" && m.scrollOffset > 0 {
				m.scrollOffset--
				return m, nil
			}
		case "pgup":
			if m.screen == screenHome && m.commandInput == "" && m.maxScrollOffset() > 0 {
				contentHeight := m.contentViewportHeight()
				m.scrollOffset += contentHeight
				m.clampScroll()
				return m, nil
			}
		case "pgdown":
			if m.screen == screenHome && m.commandInput == "" && m.scrollOffset > 0 {
				contentHeight := m.contentViewportHeight()
				m.scrollOffset -= contentHeight
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
				return m, nil
			}
		}

		switch m.screen {
		case screenHome:
			return m.updateHome(msg)
		case screenFlightForm, screenAirportForm, screenSearchForm:
			return m.updateForm(msg)
		case screenHelp:
			return m.updateHelp(msg)
		}
	}

	return m, nil
}

func (m model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "backspace":
		if m.commandInput != "" {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
			m.resetHomeInputNavigation()
		}
	case "tab":
		m.completeSlashCommand()
	case "up":
		m.previousHistory()
	case "down":
		m.nextHistory()
	case "enter":
		m.err = ""
		submittedCommand := strings.TrimSpace(m.commandInput)
		if strings.HasPrefix(submittedCommand, "/") {
			m.addHistory(submittedCommand)
		}
		q, done, err := parseSlashCommand(m.commandInput)
		if err != nil {
			return m, m.setError(err.Error())
		}
		if done {
			return m, tea.Quit
		}
		if q.kind == queryHelp {
			m.screen = screenHelp
			m.commandInput = ""
			m.statusMessage = "any key back"
			return m, nil
		}
		if q.kind == queryNone {
			m.statusMessage = "/track [flight] · /airport [code] · /search [from] [to]"
			return m, nil
		}
		m.activeRequest++
		m.loading = true
		m.statusMessage = loadingMessage(q)
		return m, tea.Batch(m.startRequest(q), spinnerTick())
	default:
		// Single-key shortcuts when command input is empty
		if m.commandInput == "" && !msg.Alt {
			switch msg.String() {
			case "t":
				m.screen = screenFlightForm
				m.inputs = []string{"", "", ""}
				m.focus = 0
				m.err = ""
				m.statusMessage = "tab next · enter submit · esc back"
				return m, nil
			case "a":
				m.screen = screenAirportForm
				m.inputs = []string{"", "", ""}
				m.focus = 0
				m.err = ""
				m.statusMessage = "tab next · enter submit · esc back"
				return m, nil
			case "s":
				m.screen = screenSearchForm
				m.inputs = []string{"", "", ""}
				m.focus = 0
				m.err = ""
				m.statusMessage = "tab next · enter submit · esc back"
				return m, nil
			case "?":
				m.screen = screenHelp
				m.statusMessage = "any key back"
				return m, nil
			}
		}
		if len(msg.Runes) > 0 && !msg.Alt {
			m.commandInput += string(msg.Runes)
			m.resetHomeInputNavigation()
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
				return m, m.setError("Flight number is required")
			}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Fetching flight status..."
			return m, tea.Batch(m.startRequest(q), spinnerTick())
		case screenAirportForm:
			code := strings.TrimSpace(m.inputs[0])
			flightType := strings.TrimSpace(m.inputs[1])
			if code == "" {
				return m, m.setError("Airport code is required")
			}
			if !airportCodePattern.MatchString(strings.ToUpper(code)) {
				return m, m.setError("Invalid airport code: use 3-letter IATA")
			}
			if flightType == "" {
				flightType = "departures"
			}
			q := query{kind: queryAirport, airport: code, flightType: flightType}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Fetching airport board..."
			return m, tea.Batch(m.startRequest(q), spinnerTick())
		case screenSearchForm:
			q := query{
				kind: querySearch,
				from: strings.TrimSpace(m.inputs[0]),
				to:   strings.TrimSpace(m.inputs[1]),
			}
			if q.from == "" || q.to == "" {
				return m, m.setError("Both airport codes are required")
			}
			if !airportCodePattern.MatchString(strings.ToUpper(q.from)) {
				return m, m.setError("Invalid origin: use 3-letter IATA")
			}
			if !airportCodePattern.MatchString(strings.ToUpper(q.to)) {
				return m, m.setError("Invalid destination: use 3-letter IATA")
			}
			m.activeRequest++
			m.loading = true
			m.err = ""
			m.statusMessage = "Searching route..."
			return m, tea.Batch(m.startRequest(q), spinnerTick())
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

func (m *model) clampScroll() {
	maxScroll := m.maxScrollOffset()
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

func (m model) maxScrollOffset() int {
	maxScroll := len(m.scrollableContentLines()) - m.contentViewportHeight()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key returns to home
	m.screen = screenHome
	m.statusMessage = "Type /help for commands"
	return m, nil
}

func (m *model) startRequest(q query) tea.Cmd {
	requestCtx, cancel := context.WithTimeout(m.appCtx, 20*time.Second)
	m.requestCancel = cancel
	return fetchQueryCmd(requestCtx, cancel, m.service, m.activeRequest, q)
}

func (m *model) setError(message string) tea.Cmd {
	m.err = message
	m.errID++
	id := m.errID
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearErrorMsg{id: id}
	})
}

func (m *model) resetHomeInputNavigation() {
	m.historyIndex = -1
	m.completionBase = ""
	m.completionMatches = nil
	m.completionIndex = -1
}

func (m *model) addHistory(command string) {
	command = strings.TrimSpace(command)
	if command == "" || !strings.HasPrefix(command, "/") {
		return
	}
	if len(m.history) > 0 && m.history[len(m.history)-1] == command {
		m.historyIndex = -1
		return
	}
	m.history = append(m.history, command)
	if len(m.history) > 10 {
		m.history = m.history[len(m.history)-10:]
	}
	m.historyIndex = -1
}

func (m *model) previousHistory() {
	if !strings.HasPrefix(m.commandInput, "/") || len(m.history) == 0 {
		return
	}
	if m.historyIndex == -1 {
		m.historyIndex = len(m.history) - 1
	} else if m.historyIndex > 0 {
		m.historyIndex--
	}
	m.commandInput = m.history[m.historyIndex]
	m.completionBase = ""
	m.completionMatches = nil
	m.completionIndex = -1
}

func (m *model) nextHistory() {
	if !strings.HasPrefix(m.commandInput, "/") || len(m.history) == 0 || m.historyIndex == -1 {
		return
	}
	if m.historyIndex < len(m.history)-1 {
		m.historyIndex++
		m.commandInput = m.history[m.historyIndex]
	} else {
		m.historyIndex = -1
		m.commandInput = ""
	}
	m.completionBase = ""
	m.completionMatches = nil
	m.completionIndex = -1
}

func (m *model) completeSlashCommand() {
	if !strings.HasPrefix(m.commandInput, "/") {
		return
	}
	m.historyIndex = -1

	prefix := commandPrefix(m.commandInput)
	if prefix == "" {
		return
	}

	activeMatch := false
	for _, match := range m.completionMatches {
		if match == prefix {
			activeMatch = true
			break
		}
	}
	canContinueCycle := activeMatch && strings.HasSuffix(m.commandInput, " ")
	if prefix != m.completionBase && !canContinueCycle {
		m.completionBase = prefix
		m.completionMatches = matchingSlashCommands(prefix)
		m.completionIndex = -1
	}
	if len(m.completionMatches) == 0 {
		return
	}

	m.completionIndex = (m.completionIndex + 1) % len(m.completionMatches)
	m.commandInput = m.completionMatches[m.completionIndex] + " "
}

func commandPrefix(input string) string {
	input = strings.TrimLeft(input, " ")
	if input == "" || !strings.HasPrefix(input, "/") {
		return ""
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return input
	}
	return fields[0]
}

func matchingSlashCommands(prefix string) []string {
	commands := []string{
		"/track",
		"/airport",
		"/search",
		"/help",
		"/quit",
		"/flight",
		"/status",
		"/board",
		"/route",
		"/exit",
	}
	prefix = strings.ToLower(prefix)
	matches := make([]string, 0, len(commands))
	for _, command := range commands {
		if strings.HasPrefix(command, prefix) {
			matches = append(matches, command)
		}
	}
	return matches
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
