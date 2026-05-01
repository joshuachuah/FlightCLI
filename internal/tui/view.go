package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/xjosh/flightcli/internal/display"
	"github.com/xjosh/flightcli/internal/models"
)

// Layout constants
const (
	statusBarHeight = 1
	inputBarHeight  = 2 // input line + keymap line
	minHeight       = 10
	minWidth        = 40
)

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return "Terminal too small. Please resize to at least 40x10."
	}

	// Calculate content area height
	contentHeight := m.height - statusBarHeight - inputBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render the three panes
	statusBar := m.renderStatusBar()
	content := m.renderContent()
	inputBar := m.renderInputBar()

	// Apply scroll offset and clip content to visible area
	contentLines := strings.Split(content, "\n")
	maxScroll := len(contentLines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}

	// Slice content based on scroll offset
	start := m.scrollOffset
	end := start + contentHeight
	if end > len(contentLines) {
		end = len(contentLines)
	}
	visibleLines := contentLines[start:end]

	// Pad to fill content area
	contentStr := strings.Join(visibleLines, "\n")
	linesNeeded := contentHeight - len(visibleLines)
	if linesNeeded > 0 {
		contentStr += strings.Repeat("\n", linesNeeded)
	}

	// Stack vertically
	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		contentStr,
		inputBar,
	)
}

func (m model) renderStatusBar() string {
	title := " FlightCLI"
	if m.activeTitle != "" {
		title = " " + m.activeTitle
	}

	titleRendered := statusBarStyle.Render(title)

	rightPart := ""
	if !m.lastUpdated.IsZero() {
		rightPart = m.lastUpdated.Format("15:04:05")
	}
	rightRendered := statusBarStyle.Render(rightPart)

	// Fill middle with spaces
	midWidth := m.width - lipgloss.Width(titleRendered) - lipgloss.Width(rightRendered)
	if midWidth < 0 {
		midWidth = 0
	}
	mid := statusBarStyle.Render(strings.Repeat(" ", midWidth))

	return lipgloss.JoinHorizontal(lipgloss.Top, titleRendered, mid, rightRendered)
}

func (m model) renderContent() string {
	switch {
	case m.loading:
		return m.renderLoading()
	case m.screen == screenHome:
		return m.viewHome()
	case m.screen == screenFlightForm:
		return m.viewFlightForm()
	case m.screen == screenAirportForm:
		return m.viewAirportForm()
	case m.screen == screenSearchForm:
		return m.viewSearchForm()
	case m.screen == screenResult:
		return m.viewResult()
	case m.screen == screenHelp:
		return m.viewHelp()
	default:
		return ""
	}
}

func (m model) renderLoading() string {
	spinner := spinnerFrames[m.spinnerFrame]
	msg := m.statusMessage
	if msg == "" {
		msg = "Loading..."
	}
	return "\n\n  " + spinnerStyle.Render(spinner) + "  " + headerStyle.Render(msg)
}

func (m model) renderInputBar() string {
	width := m.width

	if m.screen == screenHome {
		// Command input line
		prompt := inputPromptStyle.Render(" > ")
		value := inputTextStyle.Render(m.commandInput)
		cursor := inputCursorStyle.Render("▎")
		inputLine := prompt + value + cursor

		// Pad input line to full width
		inputLen := lipgloss.Width(prompt + value + cursor)
		padding := width - inputLen
		if padding < 0 {
			padding = 0
		}
		inputLine += strings.Repeat(" ", padding)

		// Keymap line
		keymap := renderKeymap([]keyHint{
			{"enter", "run"},
			{"t/a/s/?", "shortcuts"},
			{"tab", "complete"},
			{"esc", "quit"},
		})
		keymapPadding := width - lipgloss.Width(keymap)
		if keymapPadding < 0 {
			keymapPadding = 0
		}
		keymapLine := keymap + strings.Repeat(" ", keymapPadding)

		top := inputBarStyle.Render(inputLine)
		bottom := inputBarStyle.Render(keymapLine)
		return top + "\n" + bottom
	}

	// Form or result screen
	var keymapText string
	if m.screen == screenResult {
		keymapText = renderKeymap([]keyHint{
			{"r", "refresh"},
			{"↑↓", "scroll"},
			{"esc", "back"},
			{"q", "quit"},
		})
	} else if m.screen == screenHelp {
		keymapText = renderKeymap([]keyHint{
			{"esc/any", "back"},
		})
	} else {
		keymapText = renderKeymap([]keyHint{
			{"tab", "next field"},
			{"enter", "submit"},
			{"esc", "back"},
		})
	}

	inputLine := strings.Repeat(" ", width)
	keymapPadding := width - lipgloss.Width(keymapText)
	if keymapPadding < 0 {
		keymapPadding = 0
	}
	keymapLine := keymapText + strings.Repeat(" ", keymapPadding)

	top := inputBarStyle.Render(inputLine)
	bottom := inputBarStyle.Render(keymapLine)
	return top + "\n" + bottom
}

type keyHint struct {
	key  string
	desc string
}

func renderKeymap(hints []keyHint) string {
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = keyStyle.Render(h.key) + keyDescStyle.Render(":"+h.desc)
	}
	sep := keyDescStyle.Render("  ·  ")
	return " " + strings.Join(parts, sep)
}

// ── Home Screen ──────────────────────────────────────────────

func (m model) viewHome() string {
	var b strings.Builder

	// Error line
	if m.err != "" {
		b.WriteString(errorStyle.Render("  ✗ " + m.err))
		b.WriteString("\n\n")
	}

	// Welcome banner
	b.WriteString(headerStyle.Render("  FlightCLI"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Real-time flight tracking from your terminal"))
	b.WriteString("\n\n")

	// Command menu
	commands := []struct {
		cmd      string
		desc     string
		shortcut string
	}{
		{"/track AA100", "Track a flight by number", "t"},
		{"/airport JFK departures", "Show airport board", "a"},
		{"/search JFK LAX", "Search routes between airports", "s"},
		{"/help", "Show help", "?"},
		{"/quit", "Exit", ""},
	}

	for _, c := range commands {
		b.WriteString("  ")
		b.WriteString(keyStyle.Render(c.cmd))
		available := m.width - lipgloss.Width("  ") - lipgloss.Width(c.cmd) - lipgloss.Width("  ")
		if c.shortcut != "" {
			available -= lipgloss.Width("  (" + c.shortcut + ")")
		}
		if lipgloss.Width(c.desc) <= available {
			b.WriteString("  ")
			b.WriteString(keyDescStyle.Render(c.desc))
		} else {
			b.WriteString("\n    ")
			available = m.width - lipgloss.Width("    ")
			if c.shortcut != "" {
				available -= lipgloss.Width("  (" + c.shortcut + ")")
			}
			if available < 1 {
				available = 1
			}
			b.WriteString(keyDescStyle.Render(trimForWidth(c.desc, available)))
		}
		if c.shortcut != "" {
			b.WriteString("  ")
			b.WriteString(keyDescStyle.Render("(" + c.shortcut + ")"))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ── Help Screen ─────────────────────────────────────────────

func (m model) viewHelp() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  FlightCLI Help"))
	b.WriteString("\n\n")

	// Slash commands
	b.WriteString("  ")
	b.WriteString(keyStyle.Render("Commands:"))
	b.WriteString("\n")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/track [flight]", "Track a flight by number"},
		{"/airport [code]", "Show airport board (departures/arrivals)"},
		{"/search [from] [to]", "Search routes between airports"},
		{"/help", "Show this help screen"},
		{"/quit", "Exit FlightCLI"},
	}

	for _, c := range commands {
		b.WriteString("    ")
		b.WriteString(keyStyle.Render(c.cmd))
		b.WriteString("  ")
		b.WriteString(keyDescStyle.Render(c.desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Keyboard shortcuts
	b.WriteString("  ")
	b.WriteString(keyStyle.Render("Keyboard Shortcuts:"))
	b.WriteString("\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"t", "Track a flight (flight form)"},
		{"a", "Airport board (airport form)"},
		{"s", "Search routes (search form)"},
		{"?", "Show this help screen"},
		{"esc", "Go back / cancel"},
		{"q", "Quit"},
		{"enter", "Submit command or form"},
		{"tab", "Next field / complete command"},
		{"↑↓", "Scroll results / history"},
	}

	for _, s := range shortcuts {
		b.WriteString("    ")
		b.WriteString(keyStyle.Render(s.key))
		b.WriteString("  ")
		b.WriteString(keyDescStyle.Render(s.desc))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Real-time flight tracking from your terminal"))

	return b.String()
}

// ── Form Screens ─────────────────────────────────────────────

func (m model) viewFlightForm() string {
	return m.renderForm("Track Flight", []field{
		{label: "Flight number", value: strings.ToUpper(m.inputs[0]), hint: "e.g. AA100"},
	})
}

func (m model) viewAirportForm() string {
	flightType := m.inputs[1]
	if flightType == "" {
		flightType = "departures"
	}
	return m.renderForm("Airport Board", []field{
		{label: "Airport code", value: strings.ToUpper(m.inputs[0]), hint: "e.g. JFK"},
		{label: "Board type", value: strings.ToLower(flightType), hint: "departures or arrivals"},
	})
}

func (m model) viewSearchForm() string {
	return m.renderForm("Route Search", []field{
		{label: "From", value: strings.ToUpper(m.inputs[0]), hint: "e.g. JFK"},
		{label: "To", value: strings.ToUpper(m.inputs[1]), hint: "e.g. LAX"},
	})
}

type field struct {
	label string
	value string
	hint  string
}

func (m model) renderForm(title string, fields []field) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  " + title))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render("  ✗ " + m.err))
		b.WriteString("\n\n")
	}

	for i, f := range fields {
		cursor := "  "
		valueStyle := labelStyle
		if i == m.focus {
			cursor = keyStyle.Render("▸")
			valueStyle = valueStyle
		}
		b.WriteString(cursor)
		b.WriteString(" ")
		b.WriteString(labelStyle.Render(f.label + ":"))
		b.WriteString(" ")

		displayVal := f.value
		if i == m.focus {
			displayVal = valueStyle.Render(f.value) + inputCursorStyle.Render("▎")
		} else if f.value != "" {
			displayVal = valueStyle.Render(f.value)
		}
		b.WriteString(displayVal)
		b.WriteString("\n")

		b.WriteString("    ")
		b.WriteString(hintStyle.Render(f.hint))
		b.WriteString("\n\n")
	}

	return b.String()
}

// ── Result Screen ────────────────────────────────────────────

func (m model) viewResult() string {
	var b strings.Builder

	// Cached badge
	if m.lastCached {
		b.WriteString(cachedStyle.Render("  ◷ cached"))
		b.WriteString("\n\n")
	}

	// Error
	if m.err != "" {
		b.WriteString(errorStyle.Render("  ✗ " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	// Render the actual data in a styled panel
	var content string
	if m.flight != nil {
		content = formatFlight(m.flight)
	} else if m.lastQuery.kind == querySearch {
		content = formatSearchResults(m.flights)
	} else {
		content = formatBoardForWidth(m.flights, m.width)
	}

	b.WriteString(panelStyle.Render(content))
	return b.String()
}

func formatFlight(flight *models.Flight) string {
	return formatFlightAt(flight, time.Now())
}

func formatFlightAt(flight *models.Flight, now time.Time) string {
	return strings.Join(display.FlightStatusLines(flight, now), "\n")
}

func formatBoard(flights []models.AirportFlight) string {
	return formatBoardForWidth(flights, 80)
}

func formatBoardForWidth(flights []models.AirportFlight, width int) string {
	if len(flights) == 0 {
		return "No flights found."
	}

	if width <= 0 {
		width = 80
	}

	// Header row
	var header string
	switch {
	case width >= 80:
		header = tableHeaderStyle.Render(fmt.Sprintf("%-8s %-22s %-11s %-10s %s",
			"FLIGHT", "AIRLINE", "ROUTE", "STATUS", "TIME"))
	case width >= 60:
		header = tableHeaderStyle.Render(fmt.Sprintf("%-8s %-11s %-10s %s",
			"FLIGHT", "ROUTE", "STATUS", "TIME"))
	default:
		header = tableHeaderStyle.Render(fmt.Sprintf("%-8s %-11s %s",
			"FLIGHT", "ROUTE", "STATUS"))
	}

	var lines []string
	lines = append(lines, header)
	for _, f := range flights {
		scheduled := ""
		if !f.ScheduledTime.IsZero() {
			scheduled = f.ScheduledTime.Format("15:04")
		}
		statusStyled := statusStyleForFlight(f.Status).Render(trimForWidth(f.Status, 10))
		route := f.Origin + "->" + f.Destination
		switch {
		case width >= 80:
			lines = append(lines, fmt.Sprintf("%-8s %-22s %-11s %-10s %s",
				f.FlightNumber,
				trimForWidth(f.Airline, 22),
				route,
				statusStyled,
				scheduled,
			))
		case width >= 60:
			lines = append(lines, fmt.Sprintf("%-8s %-11s %-10s %s",
				f.FlightNumber,
				route,
				statusStyled,
				scheduled,
			))
		default:
			lines = append(lines, fmt.Sprintf("%-8s %-11s %s",
				f.FlightNumber,
				route,
				statusStyled,
			))
		}
	}
	return strings.Join(lines, "\n")
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

// ── Helpers ──────────────────────────────────────────────────

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
			return query{}, false, fmt.Errorf("board type must be departures or arrivals")
		}
		q := query{kind: queryAirport, airport: args[0], flightType: flightType}
		if !airportCodePattern.MatchString(strings.ToUpper(q.airport)) {
			return query{}, false, fmt.Errorf("invalid airport code %q: use a 3-letter IATA code", q.airport)
		}
		return q, false, nil
	case "search", "route":
		if len(args) != 2 {
			return query{}, false, fmt.Errorf("usage: /search JFK LAX")
		}
		q := query{kind: querySearch, from: args[0], to: args[1]}
		if !airportCodePattern.MatchString(strings.ToUpper(q.from)) {
			return query{}, false, fmt.Errorf("invalid airport code %q: use a 3-letter IATA code", q.from)
		}
		if !airportCodePattern.MatchString(strings.ToUpper(q.to)) {
			return query{}, false, fmt.Errorf("invalid airport code %q: use a 3-letter IATA code", q.to)
		}
		return q, false, nil
	case "help":
		return query{kind: queryHelp}, false, nil
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

func titleForQuery(q query) string {
	switch q.kind {
	case queryFlight:
		return "Flight " + strings.ToUpper(q.flight)
	case queryAirport:
		label := capitalize(strings.ToLower(q.flightType))
		return label + " for " + strings.ToUpper(q.airport)
	case querySearch:
		return strings.ToUpper(q.from) + " → " + strings.ToUpper(q.to)
	default:
		return "FlightCLI"
	}
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