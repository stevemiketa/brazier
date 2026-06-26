package auth_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brazier/brazier/internal/auth"
	_ "modernc.org/sqlite"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- API key tests ---

func TestAPIKeyGenerateValidate(t *testing.T) {
	store, err := auth.NewAPIKeyStore(openDB(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()

	raw, err := store.Generate(ctx, "ci-bot")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(raw, "bz_") {
		t.Errorf("key prefix wrong: %q", raw)
	}

	key, err := store.Validate(ctx, raw)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if key.Name != "ci-bot" {
		t.Errorf("name = %q", key.Name)
	}
}

func TestAPIKeyInvalid(t *testing.T) {
	store, _ := auth.NewAPIKeyStore(openDB(t))
	_, err := store.Validate(context.Background(), "bz_notakey")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestAPIKeyDelete(t *testing.T) {
	store, _ := auth.NewAPIKeyStore(openDB(t))
	ctx := context.Background()

	raw, _ := store.Generate(ctx, "tmp")
	key, _ := store.Validate(ctx, raw)
	_ = store.Delete(ctx, key.ID)

	_, err := store.Validate(ctx, raw)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestAPIKeyList(t *testing.T) {
	store, _ := auth.NewAPIKeyStore(openDB(t))
	ctx := context.Background()
	_, _ = store.Generate(ctx, "a")
	_, _ = store.Generate(ctx, "b")

	keys, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("got %d keys, want 2", len(keys))
	}
}

// --- Session tests ---

func TestSessionCreateValidate(t *testing.T) {
	store, err := auth.NewSessionStore(openDB(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()

	sess, err := store.Create(ctx, "uid-1", "octocat")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.Token == "" {
		t.Error("empty token")
	}

	got, err := store.Validate(ctx, sess.Token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.Login != "octocat" {
		t.Errorf("login = %q", got.Login)
	}
}

func TestSessionExpired(t *testing.T) {
	db := openDB(t)
	store, _ := auth.NewSessionStore(db)
	ctx := context.Background()

	// Insert an already-expired session directly.
	_, err := db.Exec(
		`INSERT INTO sessions (token, user_id, login, expires_at) VALUES (?,?,?,?)`,
		"expired-token", "u1", "user", time.Now().Add(-time.Hour).UnixMilli(),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, err = store.Validate(ctx, "expired-token")
	if err == nil {
		t.Error("expected error for expired session")
	}
}

func TestSessionDelete(t *testing.T) {
	store, _ := auth.NewSessionStore(openDB(t))
	ctx := context.Background()

	sess, _ := store.Create(ctx, "u1", "login")
	_ = store.Delete(ctx, sess.Token)

	_, err := store.Validate(ctx, sess.Token)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// --- HTTP middleware tests ---

func TestHTTPMiddlewareAPIKey(t *testing.T) {
	db := openDB(t)
	keys, _ := auth.NewAPIKeyStore(db)
	sessions, _ := auth.NewSessionStore(db)
	ctx := context.Background()

	raw, _ := keys.Generate(ctx, "test")

	called := false
	handler := auth.HTTPMiddleware(keys, sessions, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestHTTPMiddlewareUnauthorized(t *testing.T) {
	db := openDB(t)
	keys, _ := auth.NewAPIKeyStore(db)
	sessions, _ := auth.NewSessionStore(db)

	handler := auth.HTTPMiddleware(keys, sessions, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestHTTPMiddlewareSessionToken(t *testing.T) {
	db := openDB(t)
	keys, _ := auth.NewAPIKeyStore(db)
	sessions, _ := auth.NewSessionStore(db)
	ctx := context.Background()

	sess, _ := sessions.Create(ctx, "u1", "octocat")

	called := false
	handler := auth.HTTPMiddleware(keys, sessions, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+sess.Token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler not called with valid session token")
	}
}
