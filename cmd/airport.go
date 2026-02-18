/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/cache"
	"github.com/xjosh/flightcli/internal/display"
	"github.com/xjosh/flightcli/internal/provider"
	"github.com/xjosh/flightcli/internal/service"
)

var airportCmd = &cobra.Command{
	Use:   "airport [airportCode]",
	Short: "Get departures and arrivals for an airport",
	Long:  `Display departure or arrival flights for a given airport IATA code (e.g. JFK, LAX, ORD).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
		if apiKey == "" {
			printAPIKeyError()
			os.Exit(1)
		}

		airportCode := args[0]
		flightType, _ := cmd.Flags().GetString("type")

		c, _ := cache.New()
		p := &provider.AviationStackProvider{APIKey: apiKey}
		svc := service.FlightService{Provider: p, Cache: c}

		s := display.NewSpinner(fmt.Sprintf("Fetching %s for %s...", flightType, airportCode))
		s.Start()
		flights, cached, err := svc.GetAirportFlights(airportCode, flightType)
		s.Stop()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s for %s: %v\n", flightType, airportCode, err)
			os.Exit(1)
		}

		if jsonOutput {
			out, err := json.MarshalIndent(flights, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(out))
			return
		}

		display.PrintAirportFlights(flights, airportCode, flightType)
		if cached {
			display.PrintCachedIndicator()
		}
	},
}

func init() {
	rootCmd.AddCommand(airportCmd)
	airportCmd.Flags().StringP("type", "t", "departures", "Flight type: departures or arrivals")
}
