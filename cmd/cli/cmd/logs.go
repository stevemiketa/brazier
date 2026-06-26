package cmd

import (
	"fmt"
	"io"

	pb "github.com/brazier/brazier/proto/gen"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <run-id>",
		Short: "Stream logs for a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := args[0]

			conn, err := dial(masterAddr, resolveAPIKey())
			if err != nil {
				return err
			}
			defer conn.Close()

			client := pb.NewBrazierAPIClient(conn)
			stream, err := client.StreamLogs(authCtx(cmd.Context(), resolveAPIKey()), &pb.RunID{Id: runID})
			if err != nil {
				return fmt.Errorf("stream logs: %w", err)
			}

			out := cmd.OutOrStdout()
			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("recv log: %w", err)
				}
				prefix := ""
				if chunk.Stderr {
					prefix = "[stderr] "
				}
				fmt.Fprintf(out, "[%s] %s%s\n", chunk.JobId, prefix, chunk.Line)
			}
			return nil
		},
	}
}
