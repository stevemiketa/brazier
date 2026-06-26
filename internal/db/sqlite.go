package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDB is a DB implementation backed by SQLite.
type SQLiteDB struct {
	db *sql.DB
}

// OpenSQLite opens (or creates) a SQLite database at path and runs migrations.
func OpenSQLite(path string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	s := &SQLiteDB{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteDB) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS pipeline_runs (
  id         TEXT PRIMARY KEY,
  project    TEXT NOT NULL,
  state      TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS nodes (
  run_id     TEXT NOT NULL,
  node_id    TEXT NOT NULL,
  job_id     TEXT NOT NULL DEFAULT '',
  state      TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (run_id, node_id)
);

CREATE TABLE IF NOT EXISTS log_chunks (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id    TEXT NOT NULL,
  run_id    TEXT NOT NULL,
  timestamp INTEGER NOT NULL,
  line      TEXT NOT NULL,
  stderr    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_log_chunks_run_job ON log_chunks (run_id, job_id);
`)
	return err
}

// --- PipelineRun ---

func (s *SQLiteDB) CreateRun(ctx context.Context, run PipelineRun) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pipeline_runs (id, project, state, created_at, updated_at) VALUES (?,?,?,?,?)`,
		run.ID, run.Project, string(run.State),
		run.CreatedAt.UnixMilli(), run.UpdatedAt.UnixMilli(),
	)
	return err
}

func (s *SQLiteDB) GetRun(ctx context.Context, id string) (PipelineRun, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project, state, created_at, updated_at FROM pipeline_runs WHERE id = ?`, id)
	var r PipelineRun
	var createdMs, updatedMs int64
	if err := row.Scan(&r.ID, &r.Project, (*string)(&r.State), &createdMs, &updatedMs); err != nil {
		return PipelineRun{}, fmt.Errorf("get run %s: %w", id, err)
	}
	r.CreatedAt = time.UnixMilli(createdMs)
	r.UpdatedAt = time.UnixMilli(updatedMs)
	return r, nil
}

func (s *SQLiteDB) UpdateRunState(ctx context.Context, id string, state RunState) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pipeline_runs SET state = ?, updated_at = ? WHERE id = ?`,
		string(state), time.Now().UnixMilli(), id,
	)
	return err
}

func (s *SQLiteDB) ListRuns(ctx context.Context, project string, limit int) ([]PipelineRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project, state, created_at, updated_at FROM pipeline_runs
		 WHERE project = ? ORDER BY created_at DESC LIMIT ?`, project, limit)
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

func (s *SQLiteDB) UpsertNode(ctx context.Context, n NodeRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO nodes (run_id, node_id, job_id, state, updated_at)
		 VALUES (?,?,?,?,?)
		 ON CONFLICT(run_id, node_id) DO UPDATE SET
		   job_id = excluded.job_id,
		   state  = excluded.state,
		   updated_at = excluded.updated_at`,
		n.RunID, n.NodeID, n.JobID, string(n.State), time.Now().UnixMilli(),
	)
	return err
}

func (s *SQLiteDB) GetNode(ctx context.Context, runID, nodeID string) (NodeRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT run_id, node_id, job_id, state, updated_at FROM nodes WHERE run_id = ? AND node_id = ?`,
		runID, nodeID)
	var n NodeRecord
	var updatedMs int64
	if err := row.Scan(&n.RunID, &n.NodeID, &n.JobID, (*string)(&n.State), &updatedMs); err != nil {
		return NodeRecord{}, fmt.Errorf("get node %s/%s: %w", runID, nodeID, err)
	}
	n.UpdatedAt = time.UnixMilli(updatedMs)
	return n, nil
}

func (s *SQLiteDB) ListNodes(ctx context.Context, runID string) ([]NodeRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT run_id, node_id, job_id, state, updated_at FROM nodes WHERE run_id = ?`, runID)
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

func (s *SQLiteDB) AppendLog(ctx context.Context, chunk LogChunk) error {
	stderr := 0
	if chunk.Stderr {
		stderr = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_chunks (job_id, run_id, timestamp, line, stderr) VALUES (?,?,?,?,?)`,
		chunk.JobID, chunk.RunID, chunk.Timestamp, chunk.Line, stderr,
	)
	return err
}

func (s *SQLiteDB) GetLogs(ctx context.Context, runID, jobID string) ([]LogChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT job_id, run_id, timestamp, line, stderr FROM log_chunks
		 WHERE run_id = ? AND job_id = ? ORDER BY timestamp, id`,
		runID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []LogChunk
	for rows.Next() {
		var c LogChunk
		var stderr int
		if err := rows.Scan(&c.JobID, &c.RunID, &c.Timestamp, &c.Line, &stderr); err != nil {
			return nil, err
		}
		c.Stderr = stderr != 0
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (s *SQLiteDB) Close() error { return s.db.Close() }
