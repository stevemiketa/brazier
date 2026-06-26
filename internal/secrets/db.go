package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// DBSecretBackend stores secrets in the database, encrypted with AES-256-GCM.
// The master key is a 32-byte value provided at startup (BRAZIER_SECRET_KEY env var).
type DBSecretBackend struct {
	db  *sql.DB
	gcm cipher.AEAD
}

// NewDBSecretBackend opens (or reuses) a DB connection and initialises the
// secrets table. key must be exactly 32 bytes (AES-256).
func NewDBSecretBackend(db *sql.DB, key []byte) (*DBSecretBackend, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	b := &DBSecretBackend{db: db, gcm: gcm}
	if err := b.migrate(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *DBSecretBackend) migrate() error {
	_, err := b.db.Exec(`
CREATE TABLE IF NOT EXISTS secrets (
  name       TEXT PRIMARY KEY,
  ciphertext TEXT NOT NULL
)`)
	return err
}

func (b *DBSecretBackend) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (b *DBSecretBackend) decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	ns := b.gcm.NonceSize()
	if len(data) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	plain, err := b.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

func (b *DBSecretBackend) Get(ctx context.Context, name string) (string, error) {
	var enc string
	err := b.db.QueryRowContext(ctx, `SELECT ciphertext FROM secrets WHERE name = ?`, name).Scan(&enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("secret %q not found", name)
	}
	if err != nil {
		return "", err
	}
	return b.decrypt(enc)
}

func (b *DBSecretBackend) Set(ctx context.Context, name, value string) error {
	enc, err := b.encrypt(value)
	if err != nil {
		return err
	}
	_, err = b.db.ExecContext(ctx,
		`INSERT INTO secrets (name, ciphertext) VALUES (?,?)
		 ON CONFLICT(name) DO UPDATE SET ciphertext = excluded.ciphertext`,
		name, enc)
	return err
}

func (b *DBSecretBackend) Delete(ctx context.Context, name string) error {
	_, err := b.db.ExecContext(ctx, `DELETE FROM secrets WHERE name = ?`, name)
	return err
}
