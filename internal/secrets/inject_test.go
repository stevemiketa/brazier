package secrets_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/brazier/brazier/internal/secrets"
	pb "github.com/brazier/brazier/proto/gen"
	_ "modernc.org/sqlite"
)

func TestInjectSecrets(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	key := make([]byte, 32)
	b, _ := secrets.NewDBSecretBackend(db, key)
	ctx := context.Background()

	_ = b.Set(ctx, "DEPLOY_TOKEN", "tok-abc")
	_ = b.Set(ctx, "DB_PASSWORD", "hunter2")

	dispatch := &pb.JobDispatch{
		JobId: "j1",
		RunId: "r1",
		Spec: &pb.JobSpec{
			Commands: []string{"deploy.sh"},
			Secrets:  []string{"DEPLOY_TOKEN", "DB_PASSWORD"},
		},
	}

	if err := secrets.InjectSecrets(ctx, dispatch, b); err != nil {
		t.Fatalf("inject: %v", err)
	}

	if len(dispatch.Env) != 2 {
		t.Fatalf("env count = %d, want 2", len(dispatch.Env))
	}
	envMap := make(map[string]string)
	for _, e := range dispatch.Env {
		envMap[e.Key] = e.Value
	}
	if envMap["DEPLOY_TOKEN"] != "tok-abc" {
		t.Errorf("DEPLOY_TOKEN = %q", envMap["DEPLOY_TOKEN"])
	}
	if envMap["DB_PASSWORD"] != "hunter2" {
		t.Errorf("DB_PASSWORD = %q", envMap["DB_PASSWORD"])
	}
}

func TestInjectSecretsMissing(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	key := make([]byte, 32)
	b, _ := secrets.NewDBSecretBackend(db, key)

	dispatch := &pb.JobDispatch{
		Spec: &pb.JobSpec{Secrets: []string{"MISSING_SECRET"}},
	}
	if err := secrets.InjectSecrets(context.Background(), dispatch, b); err == nil {
		t.Error("expected error for missing secret")
	}
}
