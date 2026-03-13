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
		flightNumber := args[0]
		svc := newFlightService(requireAPIKey(), true)

		s := display.NewSpinner(fmt.Sprintf("Fetching status for %s...", flightNumber))
		s.Start()
		flight, cached, err := svc.GetStatus(flightNumber)
		s.Stop()

		if err != nil {
			cobra.CheckErr(fmt.Errorf("fetching status for flight %s: %w", flightNumber, err))
		}

		if jsonOutput {
			printJSONOutput(flight)
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
