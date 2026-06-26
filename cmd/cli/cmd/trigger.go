package cmd

import (
	"fmt"

	pb "github.com/brazier/brazier/proto/gen"
	"github.com/spf13/cobra"
)

func newTriggerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trigger <project>",
		Short: "Manually trigger a pipeline run for a project on master",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// A manual trigger submits an empty PipelineSpec tagged with the project name.
			// The master will fetch the project's Brazierfile and execute it server-side.
			spec := &pb.PipelineSpec{
				Workflow: &pb.WorkflowRef{Name: args[0]},
			}

			runID, err := submitPipeline(cmd.Context(), spec)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Triggered run: %s\n", runID.Id)
			return nil
		},
	}
}
