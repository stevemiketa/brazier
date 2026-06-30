// Package main is the entrypoint for the Brazier master service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brazier/brazier/internal/api"
	"github.com/brazier/brazier/internal/auth"
	"github.com/brazier/brazier/internal/bus"
	"github.com/brazier/brazier/internal/config"
	"github.com/brazier/brazier/internal/db"
	"github.com/brazier/brazier/internal/pipeline"
	"github.com/brazier/brazier/internal/registry"
	"github.com/brazier/brazier/internal/webhook"
	pb "github.com/brazier/brazier/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- Configuration ---
	configPath := os.Getenv("BRAZIER_CONFIG")
	if configPath == "" {
		configPath = "brazier.yaml"
	}
	cfg, err := config.LoadOptional(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Database ---
	// DATABASE_URL carries credentials, so it stays an env var rather than
	// living in the config file.
	var store db.DB
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		store, err = db.OpenPostgres(dbURL)
	} else {
		store, err = db.OpenSQLite(cfg.DB.SQLitePath)
	}
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	// Disable auth for local development.
	auth.AuthEnabled = false

	// --- Core subsystems ---
	eventBus := bus.New()
	reg := registry.New()

	// Adapter: pipeline.Dispatcher → registry.Registry
	dispatcher := &registryDispatcher{reg: reg}

	sched := pipeline.NewScheduler(store, dispatcher, eventBus)
	mgr := pipeline.NewManager(store, eventBus, sched)

	// --- gRPC server ---
	grpcPort := cfg.Server.GRPCPort
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterBrazierAPIServer(grpcSrv, api.NewBrazierServer(store, mgr, reg))
	pb.RegisterAgentServiceServer(grpcSrv, api.NewAgentServer(reg, store))

	// --- HTTP server (webhook + future web static) ---
	httpPort := cfg.Server.HTTPPort

	// GITHUB_WEBHOOK_SECRET is a credential, so it stays an env var.
	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	mux := http.NewServeMux()
	mux.Handle("/webhook", webhook.New(webhookSecret, func(e webhook.TriggerEvent) {
		rc := pipeline.RunContext{Branch: e.Branch, Tag: e.Tag, Event: e.Event}
		if _, err := mgr.Start(ctx, &pb.PipelineSpec{}, e.Project, rc); err != nil {
			slog.Error("start pipeline from webhook", "err", err, "project", e.Project)
		}
	}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpSrv := &http.Server{Addr: ":" + httpPort, Handler: mux}

	// --- Start ---
	errCh := make(chan error, 2)

	go func() {
		slog.Info("gRPC listening", "port", grpcPort)
		errCh <- grpcSrv.Serve(lis)
	}()
	go func() {
		slog.Info("HTTP listening", "port", httpPort)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		grpcSrv.GracefulStop()
		_ = httpSrv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
	return nil
}

// registryDispatcher adapts registry.Registry to pipeline.Dispatcher.
type registryDispatcher struct {
	reg *registry.Registry
}

func (d *registryDispatcher) Dispatch(ctx context.Context, runID, nodeID string, spec *pb.JobSpec) error {
	return d.reg.Dispatch(ctx, &pb.JobDispatch{
		JobId: nodeID,
		RunId: runID,
		Spec:  spec,
	})
}
