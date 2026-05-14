package tui

import "github.com/charmbracelet/lipgloss"

// Color palette — aviation-inspired
var (
	colorPrimary   = lipgloss.Color("#7DC8E4") // Sky blue
	colorAccent    = lipgloss.Color("#F5A623") // Amber/gold
	colorDim       = lipgloss.Color("#626262") // Muted grey
	colorError     = lipgloss.Color("#FF6B6B") // Soft red
	colorSuccess   = lipgloss.Color("#7DC896") // Soft green
	colorDark      = lipgloss.Color("#1A1A2E") // Dark navy background
	colorSurface   = lipgloss.Color("#16213E") // Slightly lighter navy
	colorBorder    = lipgloss.Color("#3A3A5C") // Muted purple border
	colorHighlight = lipgloss.Color("#E0E0E0") // Bright text
	colorMuted     = lipgloss.Color("#8888AA") // Muted text
)

var (
	// Status bar at the top of the screen
	statusBarStyle = lipgloss.NewStyle().
			Height(1).
			Padding(0, 1).
			Background(colorDark).
			Foreground(colorPrimary).
			Bold(true)

	// Content area — scrollable middle pane
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(colorHighlight)

	// Input bar at the bottom
	inputBarStyle = lipgloss.NewStyle().
			Height(1).
			Padding(0, 1).
			Background(colorDark).
			Foreground(colorHighlight)

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	inputTextStyle = lipgloss.NewStyle().
			Foreground(colorHighlight)

	inputCursorStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	// Keymap hint style (shown in input bar)
	keyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	keyDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Error messages
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	// Section headers inside content
	headerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// Labels and secondary text
	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Values (user input)
	valueStyle = lipgloss.NewStyle().
			Foreground(colorHighlight)

	// Hint text
	hintStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Cached result badge
	cachedStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// Table header for flight boards
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// Flight status colors by state
	statusOnTime    = lipgloss.NewStyle().Foreground(colorSuccess)
	statusDelayed   = lipgloss.NewStyle().Foreground(colorAccent)
	statusCancelled = lipgloss.NewStyle().Foreground(colorError)
	statusLanded    = lipgloss.NewStyle().Foreground(colorMuted)
	statusDefault   = lipgloss.NewStyle().Foreground(colorHighlight)

	// Bordered panel for results
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Spinning indicator
	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorAccent)
)

// statusStyleForFlight returns the appropriate style for a flight status string.
func statusStyleForFlight(status string) lipgloss.Style {
	switch status {
	case "On Time", "Scheduled", "In Flight":
		return statusOnTime
	case "Delayed":
		return statusDelayed
	case "Cancelled", "Diverted":
		return statusCancelled
	case "Landed", "Arrived":
		return statusLanded
	default:
		return statusDefault
	}
}
