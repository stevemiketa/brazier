// Package auth provides API key management, GitHub OAuth2, session tokens,
// and gRPC/HTTP middleware for the master service.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	apiKeyPrefix  = "bz_"
	apiKeyRawLen  = 32 // bytes of random data → 64 hex chars
	bcryptCost    = 12
)

// APIKey is the persisted record of an API key.
type APIKey struct {
	ID        string
	Name      string    // human-readable label
	Hash      string    // bcrypt hash of the raw key
	CreatedAt time.Time
}

// APIKeyStore manages API keys in the database.
type APIKeyStore struct {
	db *sql.DB
}

// NewAPIKeyStore initialises the api_keys table and returns a store.
func NewAPIKeyStore(db *sql.DB) (*APIKeyStore, error) {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS api_keys (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  hash       TEXT NOT NULL,
  created_at INTEGER NOT NULL
)`)
	if err != nil {
		return nil, fmt.Errorf("migrate api_keys: %w", err)
	}
	return &APIKeyStore{db: db}, nil
}

// Generate creates a new API key, stores its bcrypt hash, and returns the
// raw key string (shown once; not stored).
func (s *APIKeyStore) Generate(ctx context.Context, name string) (raw string, err error) {
	buf := make([]byte, apiKeyRawLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	raw = apiKeyPrefix + hex.EncodeToString(buf)

	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash key: %w", err)
	}

	id := newID()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, name, hash, created_at) VALUES (?,?,?,?)`,
		id, name, string(hash), time.Now().UnixMilli(),
	)
	if err != nil {
		return "", fmt.Errorf("store key: %w", err)
	}
	return raw, nil
}

// Validate checks whether raw matches any stored API key hash.
// Returns the matching APIKey or an error if not found / invalid.
func (s *APIKeyStore) Validate(ctx context.Context, raw string) (APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, hash, created_at FROM api_keys`)
	if err != nil {
		return APIKey{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var k APIKey
		var createdMs int64
		if err := rows.Scan(&k.ID, &k.Name, &k.Hash, &createdMs); err != nil {
			return APIKey{}, err
		}
		k.CreatedAt = time.UnixMilli(createdMs)
		if err := bcrypt.CompareHashAndPassword([]byte(k.Hash), []byte(raw)); err == nil {
			return k, nil
		}
	}
	return APIKey{}, errors.New("invalid api key")
}

// Delete removes an API key by ID.
func (s *APIKeyStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

// List returns all API keys (hashes omitted for safety).
func (s *APIKeyStore) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, created_at FROM api_keys ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var createdMs int64
		if err := rows.Scan(&k.ID, &k.Name, &createdMs); err != nil {
			return nil, err
		}
		k.CreatedAt = time.UnixMilli(createdMs)
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
