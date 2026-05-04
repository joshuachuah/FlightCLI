/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/joshuachuah/flightcli/internal/display"
)

var (
	searchFrom string
	searchTo   string
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search flights between two airports",
	Long:  `Search for current flights on a specific route using IATA airport codes.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, err := requireAPIKey()
		if err != nil {
			printAPIKeyError()
			cobra.CheckErr(err)
		}

		from, err := normalizeAirportCode(searchFrom, "--from")
		cobra.CheckErr(err)
		to, err := normalizeAirportCode(searchTo, "--to")
		cobra.CheckErr(err)
		svc := newFlightService(apiKey, true)

		s := display.NewSpinner(fmt.Sprintf("Searching flights from %s to %s...", from, to))
		s.Start()
		flights, cached, err := svc.SearchFlights(cmd.Context(), from, to)
		s.Stop()

		if err != nil {
			cobra.CheckErr(fmt.Errorf("searching flights from %s to %s: %w", from, to, err))
		}

		if jsonOutput {
			cobra.CheckErr(printJSONOutput(flights))
			return
		}

		display.PrintSearchResults(flights, from, to)
		if cached {
			display.PrintCachedIndicator()
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVar(&searchFrom, "from", "", "Departure airport IATA code (e.g. JFK)")
	searchCmd.Flags().StringVar(&searchTo, "to", "", "Arrival airport IATA code (e.g. LAX)")
	searchCmd.MarkFlagRequired("from")
	searchCmd.MarkFlagRequired("to")
}
