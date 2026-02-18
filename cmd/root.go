/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "flightcli",
	Short: "Track live flights and airport departures/arrivals",
	Long: `flightcli is a command-line tool for tracking live flight data powered
by the AviationStack API. It lets you check flight status, view airport
departure and arrival boards, track flights in real-time, and search
routes between airports.

Requires an AviationStack API key set via the AVIATIONSTACK_API_KEY
environment variable or a .env file in the current directory.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
}
