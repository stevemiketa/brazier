// Package api implements the BrazierAPI and AgentService gRPC servers.
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/brazier/brazier/internal/db"
	"github.com/brazier/brazier/internal/pipeline"
	"github.com/brazier/brazier/internal/registry"
	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BrazierServer implements pb.BrazierAPIServer.
type BrazierServer struct {
	pb.UnimplementedBrazierAPIServer
	store    db.DB
	manager  *pipeline.Manager
	reg      *registry.Registry
}

// NewBrazierServer returns a BrazierServer.
func NewBrazierServer(store db.DB, mgr *pipeline.Manager, reg *registry.Registry) *BrazierServer {
	return &BrazierServer{store: store, manager: mgr, reg: reg}
}

func (s *BrazierServer) SubmitPipeline(ctx context.Context, spec *pb.PipelineSpec) (*pb.RunID, error) {
	id, err := s.manager.Start(ctx, spec, "api", pipeline.RunContext{Event: "manual"})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "start pipeline: %v", err)
	}
	return &pb.RunID{Id: id}, nil
}

func (s *BrazierServer) GetRun(ctx context.Context, req *pb.RunID) (*pb.RunStatus, error) {
	run, err := s.store.GetRun(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "run not found: %v", err)
	}
	nodes, _ := s.store.ListNodes(ctx, req.Id)
	nodeIDs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.NodeID)
	}
	return &pb.RunStatus{
		RunId: run.ID,
		State: string(run.State),
		Nodes: nodeIDs,
	}, nil
}

func (s *BrazierServer) ListRuns(ctx context.Context, req *pb.ListRunsRequest) (*pb.RunList, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}
	runs, err := s.store.ListRuns(ctx, req.Project, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list runs: %v", err)
	}
	items := make([]*pb.RunStatus, 0, len(runs))
	for _, r := range runs {
		items = append(items, &pb.RunStatus{RunId: r.ID, State: string(r.State)})
	}
	return &pb.RunList{Runs: items}, nil
}

func (s *BrazierServer) CancelRun(ctx context.Context, req *pb.RunID) (*pb.Empty, error) {
	if err := s.store.UpdateRunState(ctx, req.Id, db.RunStateCancelled); err != nil {
		return nil, status.Errorf(codes.Internal, "cancel run: %v", err)
	}
	return &pb.Empty{}, nil
}

func (s *BrazierServer) StreamLogs(req *pb.RunID, stream pb.BrazierAPI_StreamLogsServer) error {
	ctx := stream.Context()
	var seen int
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			chunks, err := s.store.GetLogs(ctx, req.Id, "")
			if err != nil {
				return status.Errorf(codes.Internal, "get logs: %v", err)
			}
			for i := seen; i < len(chunks); i++ {
				c := chunks[i]
				if err := stream.Send(&pb.LogChunk{
					JobId:     c.JobID,
					RunId:     c.RunID,
					Timestamp: c.Timestamp,
					Line:      c.Line,
					Stderr:    c.Stderr,
				}); err != nil {
					return err
				}
			}
			seen = len(chunks)
			// Stop streaming when run reaches terminal state.
			run, _ := s.store.GetRun(ctx, req.Id)
			if run.State == db.RunStateSuccess || run.State == db.RunStateFailed || run.State == db.RunStateCancelled {
				return nil
			}
		}
	}
}

func (s *BrazierServer) ListAgents(ctx context.Context, _ *pb.Empty) (*pb.AgentList, error) {
	return &pb.AgentList{Agents: s.reg.List()}, nil
}

func (s *BrazierServer) ListWorkflows(_ context.Context, _ *pb.Empty) (*pb.WorkflowList, error) {
	return &pb.WorkflowList{}, nil
}

func (s *BrazierServer) GetWorkflow(_ context.Context, ref *pb.WorkflowRef) (*pb.WorkflowDAG, error) {
	return nil, status.Errorf(codes.Unimplemented, fmt.Sprintf("workflow %s not found", ref.Name))
}

// AgentServer implements pb.AgentServiceServer.
type AgentServer struct {
	pb.UnimplementedAgentServiceServer
	reg   *registry.Registry
	store db.DB
}

// NewAgentServer returns an AgentServer.
func NewAgentServer(reg *registry.Registry, store db.DB) *AgentServer {
	return &AgentServer{reg: reg, store: store}
}

func (s *AgentServer) Register(reg *pb.AgentRegistration, stream pb.AgentService_RegisterServer) error {
	return s.reg.Register(reg, stream)
}

func (s *AgentServer) SendLog(stream pb.AgentService_SendLogServer) error {
	ctx := stream.Context()
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return nil //nolint:nilerr // EOF is expected
		}
		_ = s.store.AppendLog(ctx, db.LogChunk{
			JobID:     chunk.JobId,
			RunID:     chunk.RunId,
			Timestamp: chunk.Timestamp,
			Line:      chunk.Line,
			Stderr:    chunk.Stderr,
		})
	}
}

func (s *AgentServer) SendResult(ctx context.Context, result *pb.JobResult) (*pb.Empty, error) {
	s.reg.JobDone(result.JobId)
	return &pb.Empty{}, nil
}
