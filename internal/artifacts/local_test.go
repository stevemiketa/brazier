package artifacts_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/brazier/brazier/internal/artifacts"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := artifacts.NewLocalStore(dir)
	ctx := context.Background()

	content := []byte("artifact contents")
	if err := store.Upload(ctx, "run-1", "bin/app", bytes.NewReader(content)); err != nil {
		t.Fatalf("upload: %v", err)
	}

	rc, err := store.Download(ctx, "run-1", "bin/app")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestLocalStoreList(t *testing.T) {
	dir := t.TempDir()
	store := artifacts.NewLocalStore(dir)
	ctx := context.Background()

	for _, path := range []string{"bin/app", "reports/coverage.out", "dist/index.js"} {
		if err := store.Upload(ctx, "run-1", path, bytes.NewReader([]byte("x"))); err != nil {
			t.Fatalf("upload %s: %v", path, err)
		}
	}

	paths, err := store.List(ctx, "run-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(paths) != 3 {
		t.Errorf("got %d paths, want 3: %v", len(paths), paths)
	}
}

func TestLocalStoreListEmpty(t *testing.T) {
	store := artifacts.NewLocalStore(t.TempDir())
	paths, err := store.List(context.Background(), "nonexistent-run")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty list, got %v", paths)
	}
}
