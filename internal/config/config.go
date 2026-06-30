// Package config loads the master/agent service configuration from a YAML
// file. It is not used to configure the brazier CLI, which is configured
// entirely via flags and environment variables (see cmd/cli).
//
// Values that are credentials or otherwise sensitive (database connection
// strings, encryption keys, webhook/OAuth secrets) are intentionally kept
// out of this struct and continue to be read directly from the environment
// at startup. Everything else — backend selection, ports, paths, non-secret
// endpoints — belongs here.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration object for the master service.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	DB        DBConfig        `yaml:"db"`
	Secrets   SecretsConfig   `yaml:"secrets"`
	Artifacts ArtifactsConfig `yaml:"artifacts"`
	Runner    RunnerConfig    `yaml:"runner"`
	Workflows WorkflowsConfig `yaml:"workflows"`
}

// ServerConfig holds the listener ports for the gRPC API and the HTTP
// (webhook + static web) server.
type ServerConfig struct {
	GRPCPort string `yaml:"grpc_port"`
	HTTPPort string `yaml:"http_port"`
}

// DBConfig selects and configures the database backend. The Postgres
// connection string itself is a credential and stays in DATABASE_URL.
type DBConfig struct {
	Backend    string `yaml:"backend"` // "sqlite" or "postgres"
	SQLitePath string `yaml:"sqlite_path"`
}

// SecretsConfig selects the secret backend and holds its non-sensitive
// configuration. Backend credentials (Vault token, cloud credentials) are
// supplied via the environment or the backend's own credential chain.
type SecretsConfig struct {
	Backend string       `yaml:"backend"` // "db", "aws-ssm", "gcp", "vault"
	AWSSSM  AWSSSMConfig `yaml:"aws_ssm"`
	GCP     GCPConfig    `yaml:"gcp"`
	Vault   VaultConfig  `yaml:"vault"`
}

type AWSSSMConfig struct {
	Prefix string `yaml:"prefix"`
}

type GCPConfig struct {
	ProjectID string `yaml:"project_id"`
}

// VaultConfig holds non-secret Vault connection settings. The Vault token
// itself is a credential and is not stored here.
type VaultConfig struct {
	Addr   string `yaml:"addr"`
	Mount  string `yaml:"mount"`
	Prefix string `yaml:"prefix"`
}

// ArtifactsConfig selects the artifact storage backend and holds its
// non-sensitive configuration.
type ArtifactsConfig struct {
	Backend     string            `yaml:"backend"` // "local", "s3", "gcs", "artifactory"
	LocalPath   string            `yaml:"local_path"`
	S3          S3Config          `yaml:"s3"`
	GCS         GCSConfig         `yaml:"gcs"`
	Artifactory ArtifactoryConfig `yaml:"artifactory"`
}

type S3Config struct {
	Bucket   string `yaml:"bucket"`
	Prefix   string `yaml:"prefix"`
	Endpoint string `yaml:"endpoint"`
}

type GCSConfig struct {
	Bucket string `yaml:"bucket"`
	Prefix string `yaml:"prefix"`
}

// ArtifactoryConfig holds non-secret Artifactory connection settings. The
// access token is a credential and is not stored here.
type ArtifactoryConfig struct {
	BaseURL string `yaml:"base_url"`
}

// RunnerConfig selects the job runner and holds its non-sensitive
// configuration.
type RunnerConfig struct {
	Type  string      `yaml:"type"` // "agent", "nomad", "k8s"
	Nomad NomadConfig `yaml:"nomad"`
	K8s   K8sConfig   `yaml:"k8s"`
}

type NomadConfig struct {
	Addr string `yaml:"addr"`
}

type K8sConfig struct {
	Namespace string `yaml:"namespace"`
	Image     string `yaml:"image"`
}

// WorkflowsConfig configures the workflows repository loader.
type WorkflowsConfig struct {
	RepoURL  string `yaml:"repo_url"`
	CacheDir string `yaml:"cache_dir"`
}

// Default returns a Config populated with the same defaults the master
// service has historically fallen back to when an env var was unset.
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			GRPCPort: "9000",
			HTTPPort: "8080",
		},
		DB: DBConfig{
			Backend:    "sqlite",
			SQLitePath: "brazier.db",
		},
		Secrets: SecretsConfig{
			Backend: "db",
		},
		Artifacts: ArtifactsConfig{
			Backend:   "local",
			LocalPath: "./artifacts",
		},
		Runner: RunnerConfig{
			Type: "agent",
		},
	}
}

// Load reads and parses a YAML config file at path, merging it onto
// Default(). Fields omitted from the file keep their default value.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}

// LoadOptional behaves like Load, but returns Default() without error if
// path does not exist.
func LoadOptional(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Default(), nil
	}
	return Load(path)
}
