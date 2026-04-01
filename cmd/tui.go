package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xjosh/flightcli/internal/tui"
)

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
