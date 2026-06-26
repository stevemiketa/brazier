package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDB is a DB implementation backed by PostgreSQL.
// Configure via DATABASE_URL env var: postgres://user:pass@host/dbname?sslmode=disable
type PostgresDB struct {
	db *sql.DB
}

// OpenPostgres opens a Postgres connection using dsn and runs migrations.
func OpenPostgres(dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	p := &PostgresDB{db: db}
	if err := p.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return p, nil
}

func (p *PostgresDB) migrate() error {
	_, err := p.db.Exec(`
CREATE TABLE IF NOT EXISTS pipeline_runs (
  id         TEXT PRIMARY KEY,
  project    TEXT NOT NULL,
  state      TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS nodes (
  run_id     TEXT NOT NULL,
  node_id    TEXT NOT NULL,
  job_id     TEXT NOT NULL DEFAULT '',
  state      TEXT NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (run_id, node_id)
);

CREATE TABLE IF NOT EXISTS log_chunks (
  id        BIGSERIAL PRIMARY KEY,
  job_id    TEXT NOT NULL,
  run_id    TEXT NOT NULL,
  timestamp BIGINT NOT NULL,
  line      TEXT NOT NULL,
  stderr    BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_log_chunks_run_job ON log_chunks (run_id, job_id);
`)
	return err
}

// --- PipelineRun ---

func (p *PostgresDB) CreateRun(ctx context.Context, run PipelineRun) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (id, project, state, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)`,
		run.ID, run.Project, string(run.State),
		run.CreatedAt.UnixMilli(), run.UpdatedAt.UnixMilli(),
	)
	return err
}

func (p *PostgresDB) GetRun(ctx context.Context, id string) (PipelineRun, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, project, state, created_at, updated_at FROM pipeline_runs WHERE id = $1`, id)
	var r PipelineRun
	var createdMs, updatedMs int64
	if err := row.Scan(&r.ID, &r.Project, (*string)(&r.State), &createdMs, &updatedMs); err != nil {
		return PipelineRun{}, fmt.Errorf("get run %s: %w", id, err)
	}
	r.CreatedAt = time.UnixMilli(createdMs)
	r.UpdatedAt = time.UnixMilli(updatedMs)
	return r, nil
}

func (p *PostgresDB) UpdateRunState(ctx context.Context, id string, state RunState) error {
	_, err := p.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = $1, updated_at = $2 WHERE id = $3`,
		string(state), time.Now().UnixMilli(), id,
	)
	return err
}

func (p *PostgresDB) ListRuns(ctx context.Context, project string, limit int) ([]PipelineRun, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, project, state, created_at, updated_at FROM pipeline_runs
		 WHERE project = $1 ORDER BY created_at DESC LIMIT $2`, project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []PipelineRun
	for rows.Next() {
		var r PipelineRun
		var createdMs, updatedMs int64
		if err := rows.Scan(&r.ID, &r.Project, (*string)(&r.State), &createdMs, &updatedMs); err != nil {
			return nil, err
		}
		r.CreatedAt = time.UnixMilli(createdMs)
		r.UpdatedAt = time.UnixMilli(updatedMs)
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// --- NodeRecord ---

func (p *PostgresDB) UpsertNode(ctx context.Context, n NodeRecord) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO nodes (run_id, node_id, job_id, state, updated_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (run_id, node_id) DO UPDATE SET
		   job_id = EXCLUDED.job_id,
		   state  = EXCLUDED.state,
		   updated_at = EXCLUDED.updated_at`,
		n.RunID, n.NodeID, n.JobID, string(n.State), time.Now().UnixMilli(),
	)
	return err
}

func (p *PostgresDB) GetNode(ctx context.Context, runID, nodeID string) (NodeRecord, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT run_id, node_id, job_id, state, updated_at FROM nodes WHERE run_id = $1 AND node_id = $2`,
		runID, nodeID)
	var n NodeRecord
	var updatedMs int64
	if err := row.Scan(&n.RunID, &n.NodeID, &n.JobID, (*string)(&n.State), &updatedMs); err != nil {
		return NodeRecord{}, fmt.Errorf("get node %s/%s: %w", runID, nodeID, err)
	}
	n.UpdatedAt = time.UnixMilli(updatedMs)
	return n, nil
}

func (p *PostgresDB) ListNodes(ctx context.Context, runID string) ([]NodeRecord, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT run_id, node_id, job_id, state, updated_at FROM nodes WHERE run_id = $1`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []NodeRecord
	for rows.Next() {
		var n NodeRecord
		var updatedMs int64
		if err := rows.Scan(&n.RunID, &n.NodeID, &n.JobID, (*string)(&n.State), &updatedMs); err != nil {
			return nil, err
		}
		n.UpdatedAt = time.UnixMilli(updatedMs)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// --- LogChunk ---

func (p *PostgresDB) AppendLog(ctx context.Context, chunk LogChunk) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO log_chunks (job_id, run_id, timestamp, line, stderr) VALUES ($1,$2,$3,$4,$5)`,
		chunk.JobID, chunk.RunID, chunk.Timestamp, chunk.Line, chunk.Stderr,
	)
	return err
}

func (p *PostgresDB) GetLogs(ctx context.Context, runID, jobID string) ([]LogChunk, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT job_id, run_id, timestamp, line, stderr FROM log_chunks
		 WHERE run_id = $1 AND job_id = $2 ORDER BY timestamp, id`,
		runID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []LogChunk
	for rows.Next() {
		var c LogChunk
		if err := rows.Scan(&c.JobID, &c.RunID, &c.Timestamp, &c.Line, &c.Stderr); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (p *PostgresDB) Close() error { return p.db.Close() }
