// Package client implements the agent-side gRPC client to the master service.
package client

import (
	"context"
	"log/slog"
	"time"

	"github.com/brazier/brazier/internal/executor"
	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient connects to the master and processes incoming jobs.
type AgentClient struct {
	masterAddr string
	reg        *pb.AgentRegistration
	exec       *executor.Executor
}

// New returns an AgentClient that will register with the master at masterAddr.
func New(masterAddr string, reg *pb.AgentRegistration, jobTimeout time.Duration) *AgentClient {
	return &AgentClient{
		masterAddr: masterAddr,
		reg:        reg,
		exec:       executor.New(jobTimeout),
	}
}

// Run connects to master, registers, and processes inbound JobDispatch messages.
// It reconnects automatically on disconnect. Blocks until ctx is cancelled.
func (a *AgentClient) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := a.connect(ctx); err != nil {
			slog.Error("agent: connection error", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (a *AgentClient) connect(ctx context.Context) error {
	conn, err := grpc.NewClient(a.masterAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	agentSvc := pb.NewAgentServiceClient(conn)

	stream, err := agentSvc.Register(ctx, a.reg)
	if err != nil {
		return err
	}
	slog.Info("agent registered", "agent_id", a.reg.AgentId, "master", a.masterAddr)

	for {
		dispatch, err := stream.Recv()
		if err != nil {
			return err
		}
		go a.handleJob(ctx, conn, dispatch)
	}
}

func (a *AgentClient) handleJob(ctx context.Context, conn *grpc.ClientConn, dispatch *pb.JobDispatch) {
	slog.Info("job received", "job_id", dispatch.JobId, "run_id", dispatch.RunId)

	agentSvc := pb.NewAgentServiceClient(conn)

	logStream, err := agentSvc.SendLog(ctx)
	if err != nil {
		slog.Error("agent: open log stream", "err", err)
	}

	sink := func(chunk *pb.LogChunk) {
		if logStream != nil {
			if err := logStream.Send(chunk); err != nil {
				slog.Warn("agent: send log chunk", "err", err)
			}
		}
	}

	success, exitCode := a.exec.Run(ctx, dispatch, sink)

	if logStream != nil {
		if _, err := logStream.CloseAndRecv(); err != nil {
			slog.Warn("agent: close log stream", "err", err)
		}
	}

	result := &pb.JobResult{
		JobId:    dispatch.JobId,
		RunId:    dispatch.RunId,
		Success:  success,
		ExitCode: int32(exitCode),
	}
	if _, err := agentSvc.SendResult(ctx, result); err != nil {
		slog.Error("agent: send result", "err", err)
	}
	slog.Info("job finished", "job_id", dispatch.JobId, "success", success, "exit_code", exitCode)
}
