package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOptional_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadOptional(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("LoadOptional: %v", err)
	}
	if cfg.Server.GRPCPort != "9000" {
		t.Errorf("GRPCPort = %q, want 9000", cfg.Server.GRPCPort)
	}
	if cfg.DB.Backend != "sqlite" {
		t.Errorf("DB.Backend = %q, want sqlite", cfg.DB.Backend)
	}
}

func TestLoad_OverridesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brazier.yaml")
	yaml := []byte(`
server:
  grpc_port: "9100"
db:
  backend: postgres
artifacts:
  backend: s3
  s3:
    bucket: my-artifacts
    prefix: ci/
runner:
  type: k8s
  k8s:
    namespace: brazier
    image: golang:1.25
workflows:
  repo_url: https://github.com/example/workflows.git
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.GRPCPort != "9100" {
		t.Errorf("GRPCPort = %q, want 9100", cfg.Server.GRPCPort)
	}
	if cfg.Server.HTTPPort != "8080" {
		t.Errorf("HTTPPort = %q, want default 8080", cfg.Server.HTTPPort)
	}
	if cfg.DB.Backend != "postgres" {
		t.Errorf("DB.Backend = %q, want postgres", cfg.DB.Backend)
	}
	if cfg.Artifacts.S3.Bucket != "my-artifacts" {
		t.Errorf("Artifacts.S3.Bucket = %q, want my-artifacts", cfg.Artifacts.S3.Bucket)
	}
	if cfg.Runner.K8s.Namespace != "brazier" {
		t.Errorf("Runner.K8s.Namespace = %q, want brazier", cfg.Runner.K8s.Namespace)
	}
	if cfg.Workflows.RepoURL != "https://github.com/example/workflows.git" {
		t.Errorf("Workflows.RepoURL = %q", cfg.Workflows.RepoURL)
	}
}
