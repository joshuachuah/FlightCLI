/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>

*/

package cmd

import (
	"fmt"
	"os"

	"github.com/xjosh/flightcli/internal/provider"
	"github.com/xjosh/flightcli/internal/service"

	"github.com/spf13/cobra"
)

var airportCmd = &cobra.Command{
	Use:   "airport [airportCode]",
	Short: "Get departures and arrivals for an airport",
	Long:  `Display departure or arrival flights for a given airport IATA code (e.g. JFK, LAX, ORD).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
		if apiKey == "" {
			fmt.Println("Error: AVIATIONSTACK_API_KEY environment variable is not set")
			return
		}

		airportCode := args[0]
		flightType, _ := cmd.Flags().GetString("type")

		p := &provider.AviationStackProvider{APIKey: apiKey}
		svc := service.FlightService{Provider: p}

		flights, err := svc.GetAirportFlights(airportCode, flightType)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		label := "Departures"
		if flightType == "arrivals" {
			label = "Arrivals"
		}
		fmt.Printf("%s for %s:\n\n", label, airportCode)

		for _, f := range flights {
			timeStr := ""
			if !f.ScheduledTime.IsZero() {
				timeStr = f.ScheduledTime.Format("15:04")
			}

			route := fmt.Sprintf("%s → %s", f.Origin, f.Destination)
			fmt.Printf("  %-10s %-25s %-14s %-12s %s\n", f.FlightNumber, f.Airline, route, f.Status, timeStr)
		}
	},
}

func init() {
	rootCmd.AddCommand(airportCmd)
	airportCmd.Flags().StringP("type", "t", "departures", "Flight type: departures or arrivals")
}
