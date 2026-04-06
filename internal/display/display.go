/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package display

import (
	"fmt"
	"math"
	"os"
	"strings"
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
	fmt.Printf("%s -> %s\n", flight.Departure, flight.Arrival)

	labelStyle.Print("Status:   ")
	StatusColor(flight.Status).Println(flight.Status)

	if !flight.DepartureTime.IsZero() {
		labelStyle.Print("Departure:")
		fmt.Printf(" %s\n", formatFlightTimestamp(flight.DepartureTime))
	}

	if !flight.ArrivalTime.IsZero() {
		labelStyle.Print("Arrival:  ")
		fmt.Printf(" %s\n", formatFlightTimestamp(flight.ArrivalTime))
	}

	totalDuration, elapsed, remaining, hasTotal, hasElapsed, hasRemaining := flightTimingMetrics(flight.DepartureTime, flight.ArrivalTime, time.Now())
	if hasTotal {
		labelStyle.Print("Flight Time:    ")
		fmt.Println(FormatDuration(totalDuration))
	}
	if hasElapsed {
		labelStyle.Print("Time Elapsed:   ")
		fmt.Println(FormatDuration(elapsed))
	}
	if hasRemaining {
		labelStyle.Print("Time to Destination: ")
		fmt.Println(FormatDuration(remaining))
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
	printAirportFlightTable(flights, airportBoardTitle(airportCode, flightType))
}

// PrintSearchResults renders a route search result table.
func PrintSearchResults(flights []models.AirportFlight, from, to string) {
	labelStyle.Printf("Flights from %s to %s:\n\n", from, to)
	if len(flights) == 0 {
		dimStyle.Println("  No flights found.")
		return
	}

	for i, f := range flights {
		printSearchFlight(f)
		if i < len(flights)-1 {
			fmt.Println()
		}
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

// FlightStatusLines returns the plain-text flight summary used by non-colored views.
func FlightStatusLines(flight *models.Flight, now time.Time) []string {
	lines := []string{
		"Flight:   " + flight.FlightNumber,
		"Airline:  " + flight.Airline,
		fmt.Sprintf("Route:    %s -> %s", flight.Departure, flight.Arrival),
		"Status:   " + flight.Status,
	}

	if !flight.DepartureTime.IsZero() {
		lines = append(lines, "Departure: "+formatFlightTimestamp(flight.DepartureTime))
	}
	if !flight.ArrivalTime.IsZero() {
		lines = append(lines, "Arrival:   "+formatFlightTimestamp(flight.ArrivalTime))
	}

	totalDuration, elapsed, remaining, hasTotal, hasElapsed, hasRemaining := flightTimingMetrics(flight.DepartureTime, flight.ArrivalTime, now)
	if hasTotal {
		lines = append(lines, "Flight Time: "+FormatDuration(totalDuration))
	}
	if hasElapsed {
		lines = append(lines, "Time Elapsed: "+FormatDuration(elapsed))
	}
	if hasRemaining {
		lines = append(lines, "Time to Destination: "+FormatDuration(remaining))
	}

	if flight.Latitude != 0 || flight.Longitude != 0 {
		lines = append(lines,
			fmt.Sprintf("Location: %.4f, %.4f", flight.Latitude, flight.Longitude),
			fmt.Sprintf("Altitude: %.0f ft", flight.Altitude),
			fmt.Sprintf("Speed:    %.0f mph", flight.Speed),
		)
	}

	return lines
}

// SearchFlightLines returns the detailed plain-text route-search summary for one flight.
func SearchFlightLines(flight models.AirportFlight) []string {
	lines := []string{
		"Flight:    " + flight.FlightNumber,
		"Airline:   " + flight.Airline,
		fmt.Sprintf("Route:     %s -> %s", flight.Origin, flight.Destination),
		"Status:    " + flight.Status,
	}

	if !flight.DepartureTime.IsZero() {
		lines = append(lines, "Departure: "+formatFlightTimestamp(flight.DepartureTime))
	}
	if !flight.ArrivalTime.IsZero() {
		lines = append(lines, "Arrival:   "+formatFlightTimestamp(flight.ArrivalTime))
	}
	if flight.Latitude != 0 || flight.Longitude != 0 {
		lines = append(lines,
			fmt.Sprintf("Location:  %.4f, %.4f", flight.Latitude, flight.Longitude),
			fmt.Sprintf("Altitude:  %.0f ft", flight.Altitude),
			fmt.Sprintf("Speed:     %.0f mph", flight.Speed),
		)
	}

	return lines
}

func airportBoardTitle(airportCode, flightType string) string {
	ft := strings.TrimSpace(flightType)
	label := "Flights"
	switch {
	case strings.EqualFold(ft, "arrivals"), strings.EqualFold(ft, "arrival"):
		label = "Arrivals"
	case strings.EqualFold(ft, "departures"), strings.EqualFold(ft, "departure"):
		label = "Departures"
	}
	return fmt.Sprintf("%s for %s", label, airportCode)
}

func printAirportFlightTable(flights []models.AirportFlight, title string) {
	labelStyle.Printf("%s:\n\n", title)
	if len(flights) == 0 {
		dimStyle.Println("  No flights found.")
		return
	}

	for _, f := range flights {
		fmt.Println(airportFlightRow(f))
	}
}

func airportFlightRow(f models.AirportFlight) string {
	timeStr := ""
	if !f.ScheduledTime.IsZero() {
		timeStr = f.ScheduledTime.Format("15:04")
	}
	route := fmt.Sprintf("%s -> %s", f.Origin, f.Destination)
	// Color is applied to status as a trailing field to avoid ANSI codes
	// disrupting fixed-width padding on earlier columns.
	coloredStatus := StatusColor(f.Status).Sprint(f.Status)
	return fmt.Sprintf("  %-10s %-25s %-15s %s  %s",
		f.FlightNumber, f.Airline, route, coloredStatus, timeStr)
}

func printSearchFlight(f models.AirportFlight) {
	labelStyle.Print("Flight:    ")
	fmt.Println(f.FlightNumber)

	labelStyle.Print("Airline:   ")
	fmt.Println(f.Airline)

	labelStyle.Print("Route:     ")
	fmt.Printf("%s -> %s\n", f.Origin, f.Destination)

	labelStyle.Print("Status:    ")
	StatusColor(f.Status).Println(f.Status)

	if !f.DepartureTime.IsZero() {
		labelStyle.Print("Departure:")
		fmt.Printf(" %s\n", formatFlightTimestamp(f.DepartureTime))
	}
	if !f.ArrivalTime.IsZero() {
		labelStyle.Print("Arrival:  ")
		fmt.Printf(" %s\n", formatFlightTimestamp(f.ArrivalTime))
	}
	if f.Latitude != 0 || f.Longitude != 0 {
		labelStyle.Print("Location:  ")
		fmt.Printf("%.4f, %.4f\n", f.Latitude, f.Longitude)
		labelStyle.Print("Altitude:  ")
		fmt.Printf("%.0f ft\n", f.Altitude)
		labelStyle.Print("Speed:     ")
		fmt.Printf("%.0f mph\n", f.Speed)
	}
}

func formatFlightTimestamp(t time.Time) string {
	return t.Format(time.RFC1123)
}

func flightTimingMetrics(departure, arrival, now time.Time) (time.Duration, time.Duration, time.Duration, bool, bool, bool) {
	if departure.IsZero() || arrival.IsZero() || !arrival.After(departure) {
		return 0, 0, 0, false, false, false
	}

	total := arrival.Sub(departure)
	var (
		elapsed      time.Duration
		remaining    time.Duration
		hasElapsed   bool
		hasRemaining bool
	)

	if now.After(departure) {
		elapsed = now.Sub(departure)
		if elapsed > total {
			elapsed = total
		}
		hasElapsed = true
	}

	if now.After(departure) && now.Before(arrival) {
		remaining = arrival.Sub(now)
		hasRemaining = true
	}

	return total, elapsed, remaining, true, hasElapsed, hasRemaining
}
