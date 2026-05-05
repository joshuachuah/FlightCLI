package cmd

import (
	"fmt"

	"github.com/joshuachuah/flightcli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("flightcli %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
	},
}