/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/cache"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local cache",
	Long:  `Manage the local flightcli cache stored in ~/.flightcli/cache/.`,
}

var cacheCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove expired and corrupt cache entries",
	Long: `Remove all expired and corrupt cache entries from disk.
The cache stores flight data with a TTL; this command cleans up
stale entries to reclaim disk space.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := cache.New()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("opening cache: %w", err))
		}

		removed, err := c.Cleanup()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("cleaning cache: %w", err))
		}

		if removed == 0 {
			fmt.Println("No expired cache entries found.")
		} else {
			fmt.Printf("Removed %d expired cache %s.\n", removed, pluralEntry(removed))
		}
	},
}

func pluralEntry(n int) string {
	if n == 1 {
		return "entry"
	}
	return "entries"
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheCleanupCmd)
}