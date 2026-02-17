/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>

*/

package cmd

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/xjosh/flightcli/internal/provider"
	"github.com/xjosh/flightcli/internal/service"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [flightNumber]",
	Short: "Get live flight status",
	Long: `CLI to track and follow flights`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
		if apiKey == "" {
			fmt.Println("Error: AVIATIONSTACK_API_KEY environment variable is not set")
			return
		}

		flightNumber := args[0]

		p := &provider.AviationStackProvider{APIKey: apiKey}
		svc := service.FlightService{Provider: p}

		flight, err := svc.GetStatus(flightNumber)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Printf("Flight: %s\n", flight.FlightNumber)
		fmt.Printf("Airline: %s\n", flight.Airline)
		fmt.Printf("Route: %s → %s\n", flight.Departure, flight.Arrival)
		fmt.Printf("Status: %s\n", flight.Status)

		if !flight.DepartureTime.IsZero() && !flight.ArrivalTime.IsZero() {
			totalDuration := flight.ArrivalTime.Sub(flight.DepartureTime)
			fmt.Printf("Flight Time: %s\n", formatDuration(totalDuration))

			now := time.Now().UTC()
			if now.Before(flight.ArrivalTime) && now.After(flight.DepartureTime) {
				remaining := flight.ArrivalTime.Sub(now)
				fmt.Printf("Time Remaining: %s\n", formatDuration(remaining))
			}
		}

		if flight.Latitude != 0 || flight.Longitude != 0 {
			fmt.Printf("Location: %.4f, %.4f\n", flight.Latitude, flight.Longitude)
			fmt.Printf("Altitude: %.0f ft\n", flight.Altitude)
			fmt.Printf("Speed: %.0f mph\n", flight.Speed)
		}
	},

}

func formatDuration(d time.Duration) string {
	h := int(math.Floor(d.Hours()))
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
