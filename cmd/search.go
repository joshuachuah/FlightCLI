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

var (
	searchFrom string
	searchTo   string
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search flights between two airports",
	Long:  `Search for current flights on a specific route using IATA airport codes.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
		if apiKey == "" {
			printAPIKeyError()
			os.Exit(1)
		}

		c, _ := cache.New()
		p := &provider.AviationStackProvider{APIKey: apiKey}
		svc := service.FlightService{Provider: p, Cache: c}

		s := display.NewSpinner(fmt.Sprintf("Searching flights from %s to %s...", searchFrom, searchTo))
		s.Start()
		flights, cached, err := svc.SearchFlights(searchFrom, searchTo)
		s.Stop()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching flights from %s to %s: %v\n", searchFrom, searchTo, err)
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

		display.PrintSearchResults(flights, searchFrom, searchTo)
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
