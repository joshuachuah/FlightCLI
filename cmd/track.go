/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/display"
	"github.com/xjosh/flightcli/internal/provider"
	"github.com/xjosh/flightcli/internal/service"
)

var trackInterval int

var trackCmd = &cobra.Command{
	Use:   "track [flightNumber]",
	Short: "Live-track a flight, refreshing automatically",
	Long:  `Continuously poll and display live flight status, refreshing on a fixed interval. Press Ctrl+C to stop.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			fmt.Fprintln(os.Stderr, "Error: --json is not supported with track (live mode).")
			fmt.Fprintln(os.Stderr, "Use 'flightcli status --json' for a one-time JSON snapshot.")
			os.Exit(1)
		}

		apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
		if apiKey == "" {
			printAPIKeyError()
			os.Exit(1)
		}

		flightNumber := args[0]
		interval := time.Duration(trackInterval) * time.Second

		p := &provider.AviationStackProvider{APIKey: apiKey}
		svc := service.FlightService{Provider: p, Cache: nil}

		// Handle Ctrl+C gracefully
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		var activeSpinner = display.NewSpinner("")

		go func() {
			<-sig
			activeSpinner.Stop()
			fmt.Print("\r\033[K") // clear the current line
			fmt.Println("Stopped tracking.")
			os.Exit(0)
		}()

		for {
			// Clear screen and move cursor to top-left
			fmt.Print("\033[2J\033[H")

			activeSpinner = display.NewSpinner(fmt.Sprintf("Fetching status for %s...", flightNumber))
			activeSpinner.Start()
			flight, _, err := svc.GetStatus(flightNumber)
			activeSpinner.Stop()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				display.PrintFlightStatus(flight)
			}

			fmt.Printf("\nLast updated: %s\n", time.Now().Format("15:04:05"))
			display.DimPrint(fmt.Sprintf("Refreshing every %ds — Press Ctrl+C to stop", trackInterval))

			time.Sleep(interval)
		}
	},
}

func init() {
	rootCmd.AddCommand(trackCmd)
	trackCmd.Flags().IntVar(&trackInterval, "interval", 30, "Refresh interval in seconds")
}
