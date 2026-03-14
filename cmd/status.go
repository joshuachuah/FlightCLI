/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/display"
)

var statusCmd = &cobra.Command{
	Use:   "status [flightNumber]",
	Short: "Get live flight status",
	Long:  `Track a live flight by its IATA flight number (e.g. AA100, KE38).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, err := requireAPIKey()
		if err != nil {
			printAPIKeyError()
			cobra.CheckErr(err)
		}

		flightNumber := args[0]
		svc := newFlightService(apiKey, true)

		s := display.NewSpinner(fmt.Sprintf("Fetching status for %s...", flightNumber))
		s.Start()
		flight, cached, err := svc.GetStatus(cmd.Context(), flightNumber)
		s.Stop()

		if err != nil {
			cobra.CheckErr(fmt.Errorf("fetching status for flight %s: %w", flightNumber, err))
		}

		if jsonOutput {
			cobra.CheckErr(printJSONOutput(flight))
			return
		}

		display.PrintFlightStatus(flight)
		if cached {
			display.PrintCachedIndicator()
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
