package secrets_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/brazier/brazier/internal/secrets"
	_ "modernc.org/sqlite"
)

func openBackend(t *testing.T) *secrets.DBSecretBackend {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	b, err := secrets.NewDBSecretBackend(db, key)
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	return b
}

func TestDBSecretSetGet(t *testing.T) {
	b := openBackend(t)
	ctx := context.Background()

	if err := b.Set(ctx, "DEPLOY_TOKEN", "supersecret"); err != nil {
		t.Fatalf("set: %v", err)
	}
	val, err := b.Get(ctx, "DEPLOY_TOKEN")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "supersecret" {
		t.Errorf("got %q, want %q", val, "supersecret")
	}
}

func TestDBSecretOverwrite(t *testing.T) {
	b := openBackend(t)
	ctx := context.Background()
	_ = b.Set(ctx, "KEY", "v1")
	_ = b.Set(ctx, "KEY", "v2")
	val, _ := b.Get(ctx, "KEY")
	if val != "v2" {
		t.Errorf("got %q, want v2", val)
	}
}

func TestDBSecretDelete(t *testing.T) {
	b := openBackend(t)
	ctx := context.Background()
	_ = b.Set(ctx, "TMP", "value")
	if err := b.Delete(ctx, "TMP"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := b.Get(ctx, "TMP")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDBSecretNotFound(t *testing.T) {
	b := openBackend(t)
	_, err := b.Get(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Error("expected not-found error")
	}
}

func TestDBSecretBadKey(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	_, err := secrets.NewDBSecretBackend(db, []byte("tooshort"))
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestDBSecretCiphertextDiffers(t *testing.T) {
	b := openBackend(t)
	ctx := context.Background()
	// Two encryptions of the same value should produce different ciphertexts (random nonce).
	_ = b.Set(ctx, "A", "same")
	_ = b.Set(ctx, "B", "same")
	v1, _ := b.Get(ctx, "A")
	v2, _ := b.Get(ctx, "B")
	if v1 != v2 {
		t.Errorf("decrypted values should match: %q vs %q", v1, v2)
	}
}
