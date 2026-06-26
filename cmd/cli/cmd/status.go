package cmd

import (
	"fmt"

	pb "github.com/brazier/brazier/proto/gen"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <run-id>",
		Short: "Show the current state of a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := dial(masterAddr, resolveAPIKey())
			if err != nil {
				return err
			}
			defer conn.Close()

			client := pb.NewBrazierAPIClient(conn)
			status, err := client.GetRun(authCtx(cmd.Context(), resolveAPIKey()), &pb.RunID{Id: args[0]})
			if err != nil {
				return fmt.Errorf("get run: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Run:   %s\n", status.RunId)
			fmt.Fprintf(out, "State: %s\n", status.State)
			if len(status.Nodes) > 0 {
				fmt.Fprintln(out, "Nodes:")
				for _, n := range status.Nodes {
					fmt.Fprintf(out, "  - %s\n", n)
				}
			}
			return nil
		},
	}
}
