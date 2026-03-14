/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/display"
)

var airportCmd = &cobra.Command{
	Use:   "airport [airportCode]",
	Short: "Get departures and arrivals for an airport",
	Long:  `Display departure or arrival flights for a given airport IATA code (e.g. JFK, LAX, ORD).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, err := requireAPIKey()
		if err != nil {
			printAPIKeyError()
			cobra.CheckErr(err)
		}

		airportCode, err := normalizeAirportCode(args[0], "airport code")
		cobra.CheckErr(err)
		flightType, _ := cmd.Flags().GetString("type")
		flightType = strings.ToLower(strings.TrimSpace(flightType))
		if flightType != "departures" && flightType != "arrivals" {
			cobra.CheckErr(fmt.Errorf("invalid --type %q: use 'departures' or 'arrivals'", flightType))
		}

		svc := newFlightService(apiKey, true)

		s := display.NewSpinner(fmt.Sprintf("Fetching %s for %s...", flightType, airportCode))
		s.Start()
		flights, cached, err := svc.GetAirportFlights(cmd.Context(), airportCode, flightType)
		s.Stop()

		if err != nil {
			cobra.CheckErr(fmt.Errorf("fetching %s for %s: %w", flightType, airportCode, err))
		}

		if jsonOutput {
			cobra.CheckErr(printJSONOutput(flights))
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
