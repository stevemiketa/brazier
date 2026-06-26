# Brazier — Architecture & Implementation Plan

Brazier is a free, self-hostable CI engine for engineering teams. It is inspired by Jenkins and CircleCI but uses an imperative pipeline model instead of YAML-based config. Pipelines are defined in code (Go or TypeScript), compiled to a protobuf spec, and dispatched to runners by a central master service.

---

## Repository Structure

```
brazier/
├── cmd/
│   ├── master/             # Go binary entrypoint — master service
│   ├── agent/              # Go binary entrypoint — bare metal agent
│   └── cli/                # Go binary entrypoint — brazier CLI
├── internal/
│   ├── scheduler/          # DAG topological sort, job dispatch (master)
│   ├── registry/           # agent registry, gRPC stream management (master)
│   ├── workflow/           # workflow repo loader, DAG parser (master)
│   ├── pipeline/           # run state machine, persistence (master)
│   ├── webhook/            # GitHub webhook handler (master)
│   ├── secrets/            # secret backend interface + implementations
│   ├── artifacts/          # artifact backend interface + implementations
│   ├── db/                 # database layer — SQLite + Postgres
│   ├── api/                # gRPC API server (master)
│   ├── executor/           # process execution, log streaming (agent)
│   └── client/             # gRPC client to master (agent)
├── proto/                  # Canonical protobuf definitions
│   ├── pipeline.proto      # PipelineSpec, WorkflowRef, Node, JobSpec, StageSpec
│   ├── runner.proto        # JobDispatch, JobResult, LogChunk, AgentRegistration
│   └── api.proto           # gRPC service definitions
├── sdk/
│   ├── go/                 # brazier-go SDK — exported, importable by Brazierfiles
│   └── ts/                 # brazier-ts SDK — npm package
├── web/                    # TypeScript/React — web UI
└── workflows/              # Example workflows repo (separate repo in production)
```

### Go module notes
- `cmd/*` — binary entrypoints only, minimal code, import from `internal/` and `sdk/go/`
- `internal/*` — all private application logic, not importable outside this module
- `sdk/go/` — public Go module (`github.com/brazier/sdk/go`), imported by user Brazierfiles and workflow files; lives in this repo but is its own `go.mod`
- Generated protobuf stubs live alongside their `.proto` files or in a `gen/` subdirectory

---

## Core Concepts

### Workflow DAG
A reusable, named pipeline graph stored in a **dedicated workflows repository** (separate from project repos). Workflow DAGs are defined using the Go DSL (`brazier-go` SDK). Each node is either a `Job` or a `Stage`. Stages contain parallel jobs. Nodes declare dependencies via `DependsOn`. Conditional edges are supported via the Go DSL.

### Brazierfile
A per-project entrypoint (`Brazierfile.go` or `Brazierfile.ts`) that lives in a project repo. It:
1. References a named workflow DAG by name and version
2. Binds job-level config (commands, env vars, secrets, artifact paths)
3. When executed as a binary, serializes the resolved `PipelineSpec` to stdout as protobuf

### Pipeline Execution Flow
```
Git event (push/PR webhook from GitHub)
  → master receives webhook, validates signature
  → master fetches Brazierfile from project repo
  → master executes Brazierfile binary via CLI subprocess
  → Brazierfile writes PipelineSpec protobuf to stdout
  → master reads and deserializes PipelineSpec
  → master fetches named WorkflowDAG from workflows repo (by semver tag or git ref)
  → master instantiates a pipeline run (binds job configs to DAG nodes)
  → master topologically sorts DAG
  → master pushes ready jobs to agents over persistent gRPC streams
  → agents execute jobs as raw processes, stream LogChunks back to master
  → master advances DAG after each node completes
    - unrelated DAG branches continue running on failure (continue strategy)
    - conditional edges evaluated at scheduling time
  → run reaches terminal state (success / failed / cancelled)
```

---

## Tech Stack

| Component | Language / Tech |
|---|---|
| Master service | Go |
| Agent runner | Go |
| CLI | Go |
| Web app | TypeScript, React |
| API | gRPC, Protocol Buffers |
| SDK (Go) | Go module |
| SDK (TypeScript) | npm package |
| Internal event bus | In-process Go channels (no external broker) |
| Agent communication | Persistent gRPC streams (master pushes to agents) |
| Runner targets | Nomad, Kubernetes, bare metal agents (abstracted via Go interface) |
| Database | SQLite (default), PostgreSQL (configurable) |
| Artifact storage | Pluggable — local disk (default), S3-compatible, GCS, Artifactory |
| Secret management | Pluggable — encrypted in DB (default), AWS SSM, GCP Secret Manager, HashiCorp Vault |
| Auth | API keys (default) + optional OAuth2 (GitHub SSO) |
| Container runtime | None — bare metal agent runs raw process commands only |
| Git provider (v1) | GitHub only |
| Workflow versioning | Semver git tags preferred (e.g. `v1.2.0`), git ref as fallback |
| DAG failure strategy | Continue — let unrelated branches of the DAG finish |

---

## Protobuf Schema

### `proto/pipeline.proto`
The lingua franca of the system. Every SDK must serialize to this spec and write it to stdout.

```proto
syntax = "proto3";
package brazier;

// Top-level output of any Brazierfile execution
message PipelineSpec {
  WorkflowRef   workflow = 1;
  repeated Node nodes    = 2;
}

// Reference to a named workflow DAG in the workflows repo
message WorkflowRef {
  string name    = 1;
  string version = 2;  // semver tag (e.g. "v1.2.0") or git ref (e.g. "main")
}

// A node in the pipeline DAG — either a Job or a Stage
message Node {
  string          id         = 1;
  repeated string depends_on = 2;
  repeated string conditions = 3;  // evaluated at schedule time, Go DSL expressions
  oneof kind {
    JobSpec   job   = 4;
    StageSpec stage = 5;
  }
}

// A single executable unit dispatched to one agent
message JobSpec {
  repeated string commands      = 1;
  repeated EnvVar env           = 2;
  repeated string secrets       = 3;   // secret names resolved server-side
  repeated string artifact_paths = 4;  // paths to collect as artifacts after job
}

// A stage: a named set of jobs that run in parallel
message StageSpec {
  repeated Node jobs = 1;
}

message EnvVar {
  string key   = 1;
  string value = 2;
}
```

### `proto/runner.proto`
Communication between master and agents.

```proto
syntax = "proto3";
package brazier;

// Master → Agent: dispatch a job for execution
message JobDispatch {
  string          job_id   = 1;
  string          run_id   = 2;
  JobSpec         spec     = 3;
  repeated EnvVar env      = 4;  // includes resolved secrets
}

// Agent → Master: streaming log output
message LogChunk {
  string job_id    = 1;
  string run_id    = 2;
  int64  timestamp = 3;
  string line      = 4;
  bool   stderr    = 5;
}

// Agent → Master: job completion
message JobResult {
  string job_id    = 1;
  string run_id    = 2;
  bool   success   = 3;
  int32  exit_code = 4;
}

// Agent → Master: registration on connect
message AgentRegistration {
  string          agent_id = 1;
  string          name     = 2;
  repeated string labels   = 3;
  int32           capacity = 4;  // max concurrent jobs
}
```

### `proto/api.proto`
gRPC service definitions consumed by the web app and CLI.

```proto
syntax = "proto3";
package brazier;

service BrazierAPI {
  // Pipeline runs
  rpc SubmitPipeline (PipelineSpec)      returns (RunID);
  rpc GetRun         (RunID)             returns (RunStatus);
  rpc ListRuns       (ListRunsRequest)   returns (RunList);
  rpc CancelRun      (RunID)             returns (Empty);
  rpc StreamLogs     (RunID)             returns (stream LogChunk);

  // Agents
  rpc ListAgents     (Empty)             returns (AgentList);

  // Workflows
  rpc ListWorkflows  (Empty)             returns (WorkflowList);
  rpc GetWorkflow    (WorkflowRef)       returns (WorkflowDAG);
}

service AgentService {
  rpc Register  (AgentRegistration)      returns (stream JobDispatch);
  rpc SendLog   (stream LogChunk)        returns (Empty);
  rpc SendResult(JobResult)              returns (Empty);
}
```

---

## Master Service (`master/`)

### Subsystems

**Webhook handler**
- HTTP endpoint for GitHub push/PR events
- Validates HMAC signature
- Enqueues trigger event onto internal event bus

**Workflow loader**
- Clones / fetches the workflows repo via go-git
- Resolves version: semver tag preferred, git ref fallback
- Executes workflow Go files as subprocesses to extract DAG structure
- Caches parsed DAGs by version

**Brazierfile executor**
- Detects `Brazierfile.go` or `Brazierfile.ts` in project repo
- Executes as subprocess, captures stdout
- Deserializes stdout bytes as `PipelineSpec` protobuf

**Pipeline run state machine**
Per-run states: `pending → running → success | failed | cancelled`
- On failure: mark node failed, continue scheduling unblocked independent branches
- Conditional edges evaluated at scheduling time using run context (branch, event type, tag)

**Scheduler**
- Topological sort of DAG nodes
- Tracks node states, dispatches nodes whose dependencies are all in terminal state
- Dispatches via Runner interface

**Agent registry**
- Tracks connected agents and their capacity/labels
- Assigns jobs to agents (simple: least-loaded; future: label matching)

**Log aggregator**
- Collects `LogChunk` streams from agents
- Persists to DB
- Fans out to web clients via server-sent gRPC streams

**Internal event bus**
- In-process Go channels only — no external broker
- Events: `TriggerReceived`, `JobDispatched`, `JobCompleted`, `JobFailed`, `RunCompleted`

### Runner Abstraction

```go
type Runner interface {
    Dispatch(ctx context.Context, job *JobDispatch) error
    Cancel(ctx context.Context, jobID string) error
}

// Implementations:
// AgentRunner  — pushes JobDispatch over persistent gRPC stream to a connected agent
// NomadRunner  — submits Nomad batch jobs
// K8sRunner    — creates Kubernetes Job resources
```

Runner is configured server-side. SDKs and Brazierfiles have no knowledge of which runner is used.

### Database Layer

Interface-driven. Two implementations:
- `SQLiteDB` — default, zero-dependency, file-based
- `PostgresDB` — production, configured via `DATABASE_URL` env var

Tables: `pipeline_runs`, `nodes`, `log_chunks`, `agents`, `projects`, `secrets`, `api_keys`, `users`

### Artifact Storage

```go
type ArtifactStore interface {
    Upload(ctx context.Context, runID, path string, r io.Reader) error
    Download(ctx context.Context, runID, path string) (io.ReadCloser, error)
    List(ctx context.Context, runID string) ([]string, error)
}

// Implementations:
// LocalStore       — default, writes to local filesystem
// S3Store          — S3-compatible (AWS S3, MinIO, Backblaze B2)
// GCSStore         — Google Cloud Storage
// ArtifactoryStore — JFrog Artifactory
```

### Secret Management

```go
type SecretBackend interface {
    Get(ctx context.Context, name string) (string, error)
    Set(ctx context.Context, name, value string) error
    Delete(ctx context.Context, name string) error
}

// Implementations:
// DBSecretBackend      — AES-256-GCM encrypted, master key from env var BRAZIER_SECRET_KEY
// AWSSSMBackend        — AWS Systems Manager Parameter Store
// GCPSecretBackend     — GCP Secret Manager
// VaultSecretBackend   — HashiCorp Vault KV
```

### Auth

- **API keys** — default, stored hashed in DB, passed as `Authorization: Bearer <key>` header
- **OAuth2 / GitHub SSO** — optional, configured via env vars (`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`)
- Session tokens issued after OAuth2 flow, stored in DB with expiry

---

## Agent (`agent/`)

Written in Go. Connects to master on startup, holds a persistent inbound gRPC stream.

### Lifecycle
1. Agent starts, calls `AgentService.Register` with name, labels, capacity
2. Master responds with a `stream JobDispatch` — agent blocks reading from it
3. On receiving `JobDispatch`, agent spawns a goroutine to execute the job
4. Executor streams `LogChunk` messages to master via `AgentService.SendLog`
5. On completion, sends `JobResult` via `AgentService.SendResult`
6. Returns to waiting for next dispatch

### Job Executor
- Runs commands as raw OS processes (no container runtime)
- Streams stdout/stderr line by line as `LogChunk` messages
- Captures exit code, reports via `JobResult`
- Enforces configurable job timeout
- Collects artifact paths and uploads to master after job completes

---

## SDK Contract

Every language SDK must implement these primitives and serialize to `PipelineSpec` protobuf on stdout when the Brazierfile binary is executed.

| Primitive | Description |
|---|---|
| `Workflow(name, version)` | Reference a named workflow DAG |
| `Job(id, spec)` | Define a job node with commands, env, secrets, artifact paths |
| `Stage(id, jobs...)` | Define a stage node containing parallel jobs |
| `DependsOn(ids...)` | Declare node dependencies |
| `When(condition)` | Attach a conditional edge (branch, tag, event type) |
| `Run(pipeline)` | Serialize PipelineSpec to protobuf, write to stdout, exit |

### brazier-go SDK (`sdk/go/`)

```go
package main

import brazier "github.com/brazier/sdk/go"

func main() {
    p := brazier.NewPipeline(
        brazier.UseWorkflow("build-test-deploy", "v1.2.0"),

        brazier.Job("lint", brazier.JobSpec{
            Commands: []string{"go vet ./...", "golangci-lint run"},
        }),

        brazier.Stage("test",
            brazier.Job("unit", brazier.JobSpec{
                Commands: []string{"go test ./..."},
            }),
            brazier.Job("integration", brazier.JobSpec{
                Commands: []string{"go test -tags=integration ./..."},
            }),
        ).DependsOn("lint"),

        brazier.Job("build", brazier.JobSpec{
            Commands:      []string{"go build -o bin/app ./cmd/app"},
            ArtifactPaths: []string{"bin/app"},
        }).DependsOn("test"),

        brazier.Job("deploy", brazier.JobSpec{
            Commands: []string{"./scripts/deploy.sh"},
            Secrets:  []string{"DEPLOY_TOKEN"},
        }).DependsOn("build").When(brazier.OnBranch("main")),
    )

    brazier.Run(p)
}
```

### brazier-ts SDK (`sdk/ts/`)

Same contract, TypeScript syntax:

```typescript
import { pipeline, job, stage, useWorkflow, run, onBranch } from "@brazier/sdk";

const p = pipeline(
    useWorkflow("build-test-deploy", "v1.2.0"),

    job("lint", { commands: ["eslint .", "tsc --noEmit"] }),

    stage("test",
        job("unit", { commands: ["jest --testPathPattern=unit"] }),
        job("e2e",  { commands: ["playwright test"] }),
    ).dependsOn("lint"),

    job("build", {
        commands: ["npm run build"],
        artifactPaths: ["dist/"],
    }).dependsOn("test"),

    job("deploy", {
        commands: ["./scripts/deploy.sh"],
        secrets: ["DEPLOY_TOKEN"],
    }).dependsOn("build").when(onBranch("main")),
);

run(p);
```

---

## Workflow DAG Spec (Workflows Repo)

Workflows are Go files in a dedicated repo, using the `brazier-go` SDK to define the DAG structure (not job implementations — those come from the Brazierfile). Master clones this repo, executes workflow files as subprocesses, and extracts the DAG shape.

```go
// workflows/build-test-deploy.go
package main

import brazier "github.com/brazier/sdk/go"

func main() {
    wf := brazier.NewWorkflow("build-test-deploy", "v1.2.0",
        brazier.Node("lint"),
        brazier.Node("test").DependsOn("lint"),
        brazier.Node("build").DependsOn("test"),
        brazier.Node("deploy").DependsOn("build").When(brazier.OnBranch("main")),
    )

    brazier.RunWorkflow(wf)
}
```

Workflow versioning:
- Tagged releases preferred: `git tag v1.2.0 && git push --tags`
- Git ref fallback: reference by branch name or commit SHA
- Brazierfile pins version: `useWorkflow("build-test-deploy", "v1.2.0")`

---

## CLI (`cli/`)

Written in Go. Thin executor — no business logic.

### Commands

| Command | Description |
|---|---|
| `brazier run` | Execute local Brazierfile, submit PipelineSpec to master |
| `brazier trigger <project>` | Manually trigger a pipeline run on master |
| `brazier logs <run-id>` | Stream logs for a run |
| `brazier status <run-id>` | Show current run state and node statuses |
| `brazier agent start` | Start a bare metal agent and connect to master |

### Brazierfile Execution
1. Detect `Brazierfile.go` or `Brazierfile.ts` in current directory
2. Compile/execute as subprocess
3. Capture stdout (PipelineSpec protobuf bytes)
4. Send to master via `BrazierAPI.SubmitPipeline` gRPC call

---

## Web App (`web/`)

TypeScript + React. Communicates with master exclusively via gRPC-web (`connectrpc`).

### Key Views
- **Dashboard** — recent pipeline runs across projects, status overview
- **Run detail** — interactive DAG visualization, per-job log streaming, node status
- **Project settings** — webhook config, secret management, runner assignment
- **Workflow browser** — browse and inspect named workflow DAG definitions
- **Agent management** — connected agents, labels, capacity, health

---

## Implementation Phases

Work through these in order. Each phase must be independently testable before moving to the next.

### Phase 1 — Proto foundation
- [x] Define `pipeline.proto` in full (PipelineSpec, WorkflowRef, Node, JobSpec, StageSpec, EnvVar)
- [x] Define `runner.proto` in full (JobDispatch, JobResult, LogChunk, AgentRegistration)
- [x] Define `api.proto` in full (BrazierAPI service, AgentService)
- [x] Generate Go stubs (`protoc-gen-go`, `protoc-gen-go-grpc`)
- [ ] Generate TypeScript stubs (`protoc-gen-connect-es`)
- [x] Write unit tests for protobuf round-trip serialization in Go

### Phase 2 — brazier-go SDK
- [x] Implement `NewPipeline`, `UseWorkflow`, `Job`, `Stage`, `DependsOn`, `When`, `Run`
- [x] Implement `NewWorkflow`, `Node`, `RunWorkflow` (for workflows repo files)
- [x] `Run()` marshals PipelineSpec to protobuf, writes to stdout, exits 0
- [x] Implement condition primitives: `OnBranch`, `OnTag`, `OnEvent`
- [x] Write unit tests for DAG construction, dependency resolution, serialization
- [x] Write example `Brazierfile.go` and example workflow file

### Phase 3 — Master service core
- [ ] GitHub webhook HTTP handler with HMAC validation
- [ ] Brazierfile executor (subprocess, stdout capture, protobuf deserialization)
- [ ] Workflow loader (go-git clone/fetch, semver tag resolution, git ref fallback, DAG execution)
- [ ] Pipeline run state machine (pending → running → success/failed/cancelled)
- [ ] Topological sort scheduler with continue-on-failure strategy
- [ ] Conditional edge evaluation at scheduling time
- [ ] In-process event bus (Go channels)
- [ ] Database layer: interface + SQLite implementation
- [ ] Database layer: PostgreSQL implementation (configured via `DATABASE_URL`)

### Phase 4 — Agent + runner abstraction
- [ ] Agent gRPC server (`AgentService.Register` persistent stream)
- [ ] Agent registry in master (track connected agents, capacity)
- [ ] `Runner` interface
- [ ] `AgentRunner` implementation (push `JobDispatch` over persistent stream)
- [ ] `NomadRunner` stub
- [ ] `K8sRunner` stub
- [ ] Agent job executor (raw process, stdout/stderr streaming as `LogChunk`)
- [ ] Agent `JobResult` reporting
- [ ] Job timeout enforcement in agent

### Phase 5 — Artifact storage + secrets
- [ ] `ArtifactStore` interface
- [ ] `LocalStore` implementation
- [ ] `S3Store` implementation (AWS S3 + S3-compatible)
- [ ] `GCSStore` implementation
- [ ] `ArtifactoryStore` implementation
- [ ] `SecretBackend` interface
- [ ] `DBSecretBackend` implementation (AES-256-GCM, key from `BRAZIER_SECRET_KEY` env)
- [ ] `AWSSSMBackend` implementation
- [ ] `GCPSecretBackend` implementation
- [ ] `VaultSecretBackend` implementation
- [ ] Secret injection into `JobDispatch` env at dispatch time

### Phase 6 — Auth
- [ ] API key generation, hashing, storage, validation middleware
- [ ] GitHub OAuth2 flow (optional, configured via env vars)
- [ ] Session token issuance and validation
- [ ] Auth middleware for gRPC API

### Phase 7 — CLI
- [ ] `brazier run` — Brazierfile detection, execution, PipelineSpec submission
- [ ] `brazier logs` — stream logs via `BrazierAPI.StreamLogs`
- [ ] `brazier status` — poll and display run + node states
- [ ] `brazier agent start` — start bare metal agent, connect to master
- [ ] `brazier trigger` — manually trigger a named project

### Phase 8 — brazier-ts SDK
- [ ] Port all Go SDK primitives to TypeScript
- [ ] Same stdout protobuf contract via `protobufjs` or `@bufbuild/protobuf`
- [ ] Condition primitives: `onBranch`, `onTag`, `onEvent`
- [ ] Write unit tests
- [ ] Write example `Brazierfile.ts` and example workflow file

### Phase 9 — Web app
- [ ] gRPC-web client setup with `connectrpc`
- [ ] Auth (API key input, GitHub OAuth2 flow)
- [ ] Dashboard — run list with status
- [ ] Run detail — DAG visualization, live log streaming
- [ ] Project settings — webhook config, secrets
- [ ] Workflow browser
- [ ] Agent management view

---

## Configuration Reference

Master is configured via environment variables:

| Env var | Default | Description |
|---|---|---|
| `BRAZIER_DB` | `sqlite` | Database backend: `sqlite` or `postgres` |
| `DATABASE_URL` | — | Postgres connection string (when `BRAZIER_DB=postgres`) |
| `BRAZIER_SECRET_KEY` | — | AES-256 master key for DB secret backend (32 bytes, hex) |
| `BRAZIER_SECRET_BACKEND` | `db` | Secret backend: `db`, `aws-ssm`, `gcp`, `vault` |
| `BRAZIER_ARTIFACT_BACKEND` | `local` | Artifact backend: `local`, `s3`, `gcs`, `artifactory` |
| `BRAZIER_ARTIFACT_PATH` | `./artifacts` | Local artifact storage path |
| `BRAZIER_RUNNER` | `agent` | Default runner: `agent`, `nomad`, `k8s` |
| `BRAZIER_WORKFLOWS_REPO` | — | Git URL of the workflows repository |
| `GITHUB_WEBHOOK_SECRET` | — | GitHub webhook HMAC secret |
| `GITHUB_CLIENT_ID` | — | GitHub OAuth2 app client ID (optional) |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth2 app client secret (optional) |
| `BRAZIER_PORT` | `9000` | gRPC API port |
| `BRAZIER_HTTP_PORT` | `8080` | Webhook + web HTTP port |