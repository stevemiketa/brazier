// Package cmd defines all brazier CLI commands using cobra.
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	masterAddr string
	apiKey     string
)

// Root returns the root cobra command with all subcommands attached.
func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "brazier",
		Short: "Brazier CI — pipeline runner CLI",
		Long:  "brazier is the command-line interface for the Brazier CI engine.",
	}

	root.PersistentFlags().StringVar(&masterAddr, "master", "localhost:9000", "gRPC address of the master service")
	root.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication (or set BRAZIER_API_KEY)")

	root.AddCommand(
		newRunCmd(),
		newLogsCmd(),
		newStatusCmd(),
		newTriggerCmd(),
		newAgentCmd(),
	)

	return root
}
