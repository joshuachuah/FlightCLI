/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package display

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/xjosh/flightcli/internal/models"
)

var (
	labelStyle  = color.New(color.FgCyan, color.Bold)
	greenStyle  = color.New(color.FgGreen)
	blueStyle   = color.New(color.FgBlue)
	yellowStyle = color.New(color.FgYellow)
	redStyle    = color.New(color.FgRed)
	dimStyle    = color.New(color.Faint)
)

// StatusColor returns the color style for a given flight status string.
func StatusColor(status string) *color.Color {
	switch status {
	case "In Flight":
		return greenStyle
	case "Landed":
		return blueStyle
	case "Scheduled":
		return yellowStyle
	case "Cancelled", "Diverted", "Incident":
		return redStyle
	default:
		return color.New(color.Reset)
	}
}

// PrintFlightStatus renders a Flight to stdout with color and labels.
func PrintFlightStatus(flight *models.Flight) {
	labelStyle.Print("Flight:   ")
	fmt.Println(flight.FlightNumber)

	labelStyle.Print("Airline:  ")
	fmt.Println(flight.Airline)

	labelStyle.Print("Route:    ")
	fmt.Printf("%s → %s\n", flight.Departure, flight.Arrival)

	labelStyle.Print("Status:   ")
	StatusColor(flight.Status).Println(flight.Status)

	if !flight.DepartureTime.IsZero() && !flight.ArrivalTime.IsZero() {
		totalDuration := flight.ArrivalTime.Sub(flight.DepartureTime)
		labelStyle.Print("Flight Time:    ")
		fmt.Println(FormatDuration(totalDuration))

		now := time.Now().UTC()
		if now.Before(flight.ArrivalTime) && now.After(flight.DepartureTime) {
			remaining := flight.ArrivalTime.Sub(now)
			labelStyle.Print("Time Remaining: ")
			fmt.Println(FormatDuration(remaining))
		}
	}

	if flight.Latitude != 0 || flight.Longitude != 0 {
		labelStyle.Print("Location: ")
		fmt.Printf("%.4f, %.4f\n", flight.Latitude, flight.Longitude)
		labelStyle.Print("Altitude: ")
		fmt.Printf("%.0f ft\n", flight.Altitude)
		labelStyle.Print("Speed:    ")
		fmt.Printf("%.0f mph\n", flight.Speed)
	}
}

// PrintAirportFlights renders the airport flight table with colored status.
func PrintAirportFlights(flights []models.AirportFlight, airportCode string, flightType string) {
	label := "Departures"
	if flightType == "arrivals" {
		label = "Arrivals"
	}
	labelStyle.Printf("%s for %s:\n\n", label, airportCode)

	for _, f := range flights {
		timeStr := ""
		if !f.ScheduledTime.IsZero() {
			timeStr = f.ScheduledTime.Format("15:04")
		}
		route := fmt.Sprintf("%s → %s", f.Origin, f.Destination)
		// Color is applied to status as a trailing field to avoid ANSI codes
		// disrupting fixed-width padding on earlier columns.
		coloredStatus := StatusColor(f.Status).Sprint(f.Status)
		fmt.Printf("  %-10s %-25s %-15s %s  %s\n",
			f.FlightNumber, f.Airline, route, coloredStatus, timeStr)
	}
}

// PrintSearchResults renders a route search result table.
func PrintSearchResults(flights []models.AirportFlight, from, to string) {
	labelStyle.Printf("Flights from %s to %s:\n\n", from, to)

	for _, f := range flights {
		timeStr := ""
		if !f.ScheduledTime.IsZero() {
			timeStr = f.ScheduledTime.Format("15:04")
		}
		route := fmt.Sprintf("%s → %s", f.Origin, f.Destination)
		coloredStatus := StatusColor(f.Status).Sprint(f.Status)
		fmt.Printf("  %-10s %-25s %-15s %s  %s\n",
			f.FlightNumber, f.Airline, route, coloredStatus, timeStr)
	}
}

// PrintCachedIndicator prints a dim "(cached)" label on its own line.
func PrintCachedIndicator() {
	dimStyle.Println("(cached)")
}

// DimPrint prints a string in dim/faint style followed by a newline.
func DimPrint(s string) {
	dimStyle.Println(s)
}

// FormatDuration formats a duration as "Xh Ym" or "Ym".
func FormatDuration(d time.Duration) string {
	h := int(math.Floor(d.Hours()))
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// NewSpinner creates a pre-configured spinner that writes to stderr.
func NewSpinner(suffix string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " " + suffix
	return s
}
