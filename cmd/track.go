/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
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
)

var trackInterval int

var trackCmd = &cobra.Command{
	Use:   "track [flightNumber]",
	Short: "Live-track a flight, refreshing automatically",
	Long:  `Continuously poll and display live flight status, refreshing on a fixed interval. Press Ctrl+C to stop.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			cobra.CheckErr("--json is not supported with track (live mode); use 'flightcli status --json' for a snapshot")
		}
		if trackInterval <= 0 {
			cobra.CheckErr("--interval must be greater than 0 seconds")
		}

		flightNumber := args[0]
		interval := time.Duration(trackInterval) * time.Second
		svc := newFlightService(requireAPIKey(), false)
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			if err := ctx.Err(); err != nil {
				fmt.Println("Stopped tracking.")
				return
			}

			fmt.Print("\033[2J\033[H")

			s := display.NewSpinner(fmt.Sprintf("Fetching status for %s...", flightNumber))
			s.Start()
			flight, _, err := svc.GetStatus(ctx, flightNumber)
			s.Stop()

			fmt.Print("\r\033[K")
			if err != nil {
				if ctx.Err() != nil {
					fmt.Println("Stopped tracking.")
					return
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				display.PrintFlightStatus(flight)
			}

			fmt.Printf("\nLast updated: %s\n", time.Now().Format("15:04:05"))
			display.DimPrint(fmt.Sprintf("Refreshing every %ds - Press Ctrl+C to stop", trackInterval))

			select {
			case <-ctx.Done():
				fmt.Println("Stopped tracking.")
				return
			case <-ticker.C:
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(trackCmd)
	trackCmd.Flags().IntVar(&trackInterval, "interval", 30, "Refresh interval in seconds")
}
