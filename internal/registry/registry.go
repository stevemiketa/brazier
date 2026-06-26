// Package registry tracks connected agents and their capacities.
package registry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	pb "github.com/brazier/brazier/proto/gen"
)

// agentConn holds the state for one connected agent.
type agentConn struct {
	info   *pb.AgentRegistration
	stream pb.AgentService_RegisterServer
	active int32 // jobs currently running on this agent
}

// Registry tracks connected agents and implements runner.Runner by pushing
// JobDispatch messages over each agent's persistent gRPC stream.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*agentConn
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{agents: make(map[string]*agentConn)}
}

// Register adds an agent and blocks until the stream ends (client disconnect).
// This is called from the AgentService.Register gRPC handler.
func (r *Registry) Register(reg *pb.AgentRegistration, stream pb.AgentService_RegisterServer) error {
	r.mu.Lock()
	r.agents[reg.AgentId] = &agentConn{info: reg, stream: stream}
	r.mu.Unlock()

	slog.Info("agent connected", "agent_id", reg.AgentId, "name", reg.Name, "capacity", reg.Capacity)

	// Block until the client disconnects.
	<-stream.Context().Done()

	r.mu.Lock()
	delete(r.agents, reg.AgentId)
	r.mu.Unlock()

	slog.Info("agent disconnected", "agent_id", reg.AgentId)
	return nil
}

// Dispatch sends a JobDispatch to the least-loaded connected agent.
// Implements runner.Runner.
func (r *Registry) Dispatch(_ context.Context, job *pb.JobDispatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent := r.leastLoaded()
	if agent == nil {
		return fmt.Errorf("no agents available")
	}

	if err := agent.stream.Send(job); err != nil {
		return fmt.Errorf("send to agent %s: %w", agent.info.AgentId, err)
	}
	agent.active++
	slog.Info("job dispatched", "job_id", job.JobId, "agent", agent.info.AgentId)
	return nil
}

// Cancel is a no-op at the registry level; individual agents handle timeout.
func (r *Registry) Cancel(_ context.Context, jobID string) error {
	slog.Warn("cancel not yet implemented", "job_id", jobID)
	return nil
}

// leastLoaded returns the agent with the most remaining capacity, or nil if none.
// Caller must hold r.mu.
func (r *Registry) leastLoaded() *agentConn {
	var best *agentConn
	for _, a := range r.agents {
		if int(a.active) >= int(a.info.Capacity) {
			continue
		}
		if best == nil || a.active < best.active {
			best = a
		}
	}
	return best
}

// JobDone decrements the active count for the agent that held jobID.
func (r *Registry) JobDone(agentID string) {
	r.mu.Lock()
	if a, ok := r.agents[agentID]; ok && a.active > 0 {
		a.active--
	}
	r.mu.Unlock()
}

// List returns info about all connected agents.
func (r *Registry) List() []*pb.AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*pb.AgentInfo, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, &pb.AgentInfo{
			AgentId:  a.info.AgentId,
			Name:     a.info.Name,
			Labels:   a.info.Labels,
			Capacity: a.info.Capacity,
			Active:   a.active,
		})
	}
	return out
}
