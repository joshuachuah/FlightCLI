package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/joshuachuah/flightcli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive terminal UI",
	Long:  "Open the interactive terminal UI for slash-command based flight tracking, airport boards, and route search.",
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(runTUI(cmd))
	},
}

func runTUI(cmd *cobra.Command) error {
	if jsonOutput {
		return fmt.Errorf("--json is only supported with a command such as status, airport, or search")
	}

	apiKey, err := requireAPIKey()
	if err != nil {
		printAPIKeyError()
		return err
	}

	svc := newFlightService(apiKey, true)
	return tui.Launch(cmd.Context(), svc)
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
