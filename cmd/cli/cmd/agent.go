package cmd

import (
	"fmt"
	"time"

	"github.com/brazier/brazier/internal/client"
	pb "github.com/brazier/brazier/proto/gen"
	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	agent := &cobra.Command{
		Use:   "agent",
		Short: "Manage bare-metal agents",
	}
	agent.AddCommand(newAgentStartCmd())
	return agent
}

func newAgentStartCmd() *cobra.Command {
	var (
		agentName  string
		capacity   int32
		labels     []string
		jobTimeout time.Duration
	)

	c := &cobra.Command{
		Use:   "start",
		Short: "Start a bare-metal agent and connect it to master",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if agentName == "" {
				return fmt.Errorf("--name is required")
			}

			reg := &pb.AgentRegistration{
				AgentId:  newAgentID(),
				Name:     agentName,
				Labels:   labels,
				Capacity: capacity,
			}

			a := client.New(masterAddr, reg, jobTimeout)
			fmt.Fprintf(cmd.OutOrStdout(), "Agent %s connecting to %s...\n", reg.AgentId, masterAddr)
			a.Run(cmd.Context())
			return nil
		},
	}

	c.Flags().StringVar(&agentName, "name", "", "Agent name (required)")
	c.Flags().Int32Var(&capacity, "capacity", 4, "Maximum concurrent jobs")
	c.Flags().StringSliceVar(&labels, "labels", nil, "Agent labels for job matching (comma-separated)")
	c.Flags().DurationVar(&jobTimeout, "job-timeout", 30*time.Minute, "Per-job execution timeout")
	return c
}

func newAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}
