package cmd

import (
	"context"
	"fmt"

	"github.com/brazier/brazier/internal/workflow"
	pb "github.com/brazier/brazier/proto/gen"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var dir string

	c := &cobra.Command{
		Use:   "run",
		Short: "Execute the local Brazierfile and submit the pipeline to master",
		RunE: func(cmd *cobra.Command, _ []string) error {
			spec, err := workflow.ExecBrazierfile(cmd.Context(), dir)
			if err != nil {
				return fmt.Errorf("exec brazierfile: %w", err)
			}

			runID, err := submitPipeline(cmd.Context(), spec)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Run started: %s\n", runID.Id)
			return nil
		},
	}

	c.Flags().StringVarP(&dir, "dir", "d", ".", "Directory containing the Brazierfile")
	return c
}

func submitPipeline(ctx context.Context, spec *pb.PipelineSpec) (*pb.RunID, error) {
	conn, err := dial(masterAddr, resolveAPIKey())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewBrazierAPIClient(conn)
	runID, err := client.SubmitPipeline(authCtx(ctx, resolveAPIKey()), spec)
	if err != nil {
		return nil, fmt.Errorf("submit pipeline: %w", err)
	}
	return runID, nil
}
